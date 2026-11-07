package api

import (
	"coordinator/internal/app"

	"github.com/gin-gonic/gin"
)

func LoadServiceRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app))

	router.GET("/search", func(c *gin.Context) {})
	router.GET("/recommended", func(c *gin.Context) {})
	router.GET("/video/:videoId", func(c *gin.Context) {})
	router.GET("/video/:videoId/recommended", func(c *gin.Context) {})

	auth.POST("/video/:videoId/score", func(c *gin.Context) {})

	auth.GET("/video/:videoId/comments", func(c *gin.Context) {})

	auth.POST("/video/:videoId/comments", func(c *gin.Context) {})

	auth.PUT("/video/comments/:commentId", func(c *gin.Context) {})
	auth.DELETE("/video/comments/:commentId", func(c *gin.Context) {})
}
