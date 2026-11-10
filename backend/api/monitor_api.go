//go:build web

package api

import (
	"net/http"
	"tinyrdm/backend/services"

	"github.com/gin-gonic/gin"
)

func registerMonitorRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/monitor")

	g.POST("/start", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Monitor().StartMonitor(req.Server))
	})

	g.POST("/stop", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Monitor().StopMonitor(req.Server))
	})

	g.POST("/export-log", func(c *gin.Context) {
		var req struct {
			Logs []string `json:"logs"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Monitor().ExportLog(req.Logs))
	})
}
