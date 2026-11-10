//go:build web

package api

import (
	"net/http"
	"tinyrdm/backend/services"
	"tinyrdm/backend/types"

	"github.com/gin-gonic/gin"
)

func registerPreferencesRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/preferences")

	g.GET("/get", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Preferences().GetPreferences())
	})

	g.POST("/set", func(c *gin.Context) {
		var pf types.Preferences
		if !bind(c, &pf) {
			return
		}
		c.JSON(http.StatusOK, services.Preferences().SetPreferences(pf))
	})

	g.POST("/update", func(c *gin.Context) {
		var value map[string]any
		if !bind(c, &value) {
			return
		}
		c.JSON(http.StatusOK, services.Preferences().UpdatePreferences(value))
	})

	g.POST("/restore", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Preferences().RestorePreferences())
	})

	g.GET("/font-list", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Preferences().GetFontList())
	})

	g.GET("/buildin-decoder", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Preferences().GetBuildInDecoder())
	})

	g.GET("/check-update", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Preferences().CheckForUpdate())
	})
}
