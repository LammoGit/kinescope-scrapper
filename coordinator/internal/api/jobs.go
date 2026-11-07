package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"

	"github.com/gin-gonic/gin"
)

// LoadJobsRouter
func LoadJobsRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app, domain.UserRoleOperator, domain.UserRoleAdministrator))

	// List all jobs
	router.GET("", func(c *gin.Context) {})

	// Create a new job
	auth.POST("", func(c *gin.Context) {})

	// Get a job
	router.GET("/:jobId", func(c *gin.Context) {})

	// Patch a job
	router.PATCH("/:jobId", func(c *gin.Context) {})

	// Delete a job
	router.DELETE("/:jobId", func(c *gin.Context) {})

	// Start a job
	router.POST("/:jobId/start", func(c *gin.Context) {})

	// Pause a job
	router.POST("/:jobId/pause", func(c *gin.Context) {})

	// Resume a job
	router.POST("/:jobId/resume", func(c *gin.Context) {})

	// Stop a job
	router.POST("/:jobId/stop", func(c *gin.Context) {})

	// List identifiers of videos downloaded for a job
	router.GET("/:jobId/items", func(c *gin.Context) {})

	// List events for a job
	router.GET("/:jobId/events", func(c *gin.Context) {})
}
