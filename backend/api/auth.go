//go:build web

package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"tinyrdm/backend/types"

	"github.com/gin-gonic/gin"
)

// authCookieName is the cookie carrying the session token
const authCookieName = "rdm_token"

// Auth configuration
var (
	authEnabled  bool
	authUsername string
	authPassword string
	jwtSecret    []byte
	sessionTTL   = 24 * time.Hour
)

// Rate limiter for login attempts
type rateLimiter struct {
	mu          sync.Mutex
	attempts    map[string][]time.Time // ip -> timestamps
	maxRate     int                    // max attempts per window
	window      time.Duration
	maxEntries  int // max tracked IPs to prevent memory exhaustion
	lastCleanup time.Time
}

var loginLimiter = &rateLimiter{
	attempts:    make(map[string][]time.Time),
	maxRate:     5,
	window:      time.Minute,
	maxEntries:  10000,
	lastCleanup: time.Now(),
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Periodic full cleanup every 5 minutes to prevent memory leak
	if now.Sub(rl.lastCleanup) > 5*time.Minute {
		for k, times := range rl.attempts {
			valid := times[:0]
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.attempts, k)
			} else {
				rl.attempts[k] = valid
			}
		}
		rl.lastCleanup = now
	}

	// Hard cap on tracked IPs to prevent memory exhaustion from distributed attacks
	if len(rl.attempts) >= rl.maxEntries {
		if _, exists := rl.attempts[ip]; !exists {
			// Too many tracked IPs, reject new ones as a safety measure
			return false
		}
	}

	// Clean old entries for this IP
	times := rl.attempts[ip]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts[ip] = valid

	if len(valid) >= rl.maxRate {
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

// InitAuth reads auth config from environment variables
func InitAuth() {
	authUsername = os.Getenv("ADMIN_USERNAME")
	authPassword = os.Getenv("ADMIN_PASSWORD")
	authEnabled = authUsername != "" && authPassword != ""

	// Generate random JWT secret on each startup
	secret := make([]byte, 32)
	rand.Read(secret)
	jwtSecret = secret

	if ttl := os.Getenv("SESSION_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			sessionTTL = d
		}
	}

	if authEnabled {
		fmt.Printf("Auth enabled for user: %s\n", authUsername)
	} else {
		fmt.Println("Auth disabled (set ADMIN_USERNAME and ADMIN_PASSWORD to enable)")
	}
}

// Simple JWT-like token: header.payload.signature (HMAC-SHA256)
type tokenPayload struct {
	User string `json:"u"`
	Exp  int64  `json:"e"`
	IP   string `json:"ip"`
}

func generateToken(username, ip string) (string, time.Time) {
	exp := time.Now().Add(sessionTTL)
	payload := tokenPayload{User: username, Exp: exp.Unix(), IP: ip}
	data, _ := json.Marshal(payload)
	encoded := hex.EncodeToString(data)

	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write(data)
	sig := hex.EncodeToString(mac.Sum(nil))

	return encoded + "." + sig, exp
}

func validateToken(token, ip string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}

	data, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}

	// Verify signature
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write(data)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
		return false
	}

	// Parse and validate payload
	var payload tokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}

	if time.Now().Unix() > payload.Exp {
		return false
	}

	// IP binding
	if payload.IP != ip {
		return false
	}

	return true
}

func getClientIP(c *gin.Context) string {
	// Cloudflare
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return c.ClientIP()
}

// tokenFromCookie returns the session token carried by the request cookie
func tokenFromCookie(c *gin.Context) string {
	token, err := c.Cookie(authCookieName)
	if err != nil {
		return ""
	}
	return token
}

// isAuthenticated reports whether the request carries a valid session token
func isAuthenticated(c *gin.Context) bool {
	if !authEnabled {
		return true
	}
	return validateToken(tokenFromCookie(c), getClientIP(c))
}

// AuthMiddleware protects API routes
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthenticated(c) {
			fail(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		c.Next()
	}
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://static.cloudflareinsights.com https://analytics.tinycraft.cc; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' ws: wss: https://static.cloudflareinsights.com https://analytics.tinycraft.cc; font-src 'self' data:;")
		c.Next()
	}
}

// registerAuthRoutes registers login/logout/status endpoints
func registerAuthRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", handleLogin)
	r.POST("/api/auth/logout", handleLogout)
	r.GET("/api/auth/status", handleAuthStatus)
}

func handleLogin(c *gin.Context) {
	ip := getClientIP(c)

	// Rate limiting
	if !loginLimiter.allow(ip) {
		fail(c, http.StatusTooManyRequests, "too many login attempts, please try again later")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bind(c, &req) {
		return
	}

	// Constant-time comparison to prevent timing attacks
	userOK := hmac.Equal([]byte(req.Username), []byte(authUsername))
	passOK := hmac.Equal([]byte(req.Password), []byte(authPassword))

	if !userOK || !passOK {
		// Delay to slow down brute force
		time.Sleep(500 * time.Millisecond)
		fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, exp := generateToken(req.Username, ip)

	// Set httpOnly, secure cookie
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, token, int(sessionTTL.Seconds()), "/", "", secure, true)

	c.JSON(http.StatusOK, types.JSResp{
		Success: true,
		Data: gin.H{
			"expires": exp.Unix(),
		},
	})
}

func handleLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, types.JSResp{Success: true})
}

func handleAuthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, types.JSResp{
		Success: true,
		Data:    gin.H{"enabled": authEnabled, "authenticated": isAuthenticated(c)},
	})
}
