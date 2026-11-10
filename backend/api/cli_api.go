//go:build web

package api

import (
	"net/http"
	"tinyrdm/backend/services"

	"github.com/gin-gonic/gin"
)

func registerCLIRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/cli")

	g.POST("/start", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
			DB     int    `json:"db"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Cli().StartCli(req.Server, req.DB))
	})

	g.POST("/close", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Cli().CloseCli(req.Server))
	})

	// CLI input is handled via WebSocket - the frontend sends
	// {"event": "cmd:input:<server>", "data": "<command>"} over WS
}
