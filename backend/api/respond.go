//go:build web

package api

import (
	"net/http"
	"tinyrdm/backend/types"

	"github.com/gin-gonic/gin"
)

// bind decodes the JSON request body into dst, writing a 400 response on failure
func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		fail(c, http.StatusBadRequest, "invalid request")
		return false
	}
	return true
}

// fail aborts the request with a JSON envelope carrying success=false
func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, types.JSResp{Msg: msg})
}