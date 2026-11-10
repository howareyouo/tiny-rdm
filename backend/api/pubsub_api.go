//go:build web

package api

import (
	"net/http"
	"tinyrdm/backend/services"

	"github.com/gin-gonic/gin"
)

func registerPubsubRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/pubsub")

	g.POST("/publish", func(c *gin.Context) {
		var req struct {
			Server  string `json:"server"`
			Channel string `json:"channel"`
			Payload string `json:"payload"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Pubsub().Publish(req.Server, req.Channel, req.Payload))
	})

	g.POST("/subscribe", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Pubsub().StartSubscribe(req.Server))
	})

	g.POST("/unsubscribe", func(c *gin.Context) {
		var req struct {
			Server string `json:"server"`
		}
		if !bind(c, &req) {
			return
		}
		c.JSON(http.StatusOK, services.Pubsub().StopSubscribe(req.Server))
	})
}
