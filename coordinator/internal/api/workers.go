package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"

	"github.com/gin-gonic/gin"
)

func LoadWorkersRouter(router gin.IRouter, app *app.App) {
	user := router.Group("", bearerAuthMiddleware(app, domain.UserRoleAdministrator, domain.UserRoleOperator))
	worker := router.Group("", signatureAuthMiddleware(app))

	// Register a new worker
	user.POST("/register", func(c *gin.Context) {})

	// Heartbeat to prolong worker-coordinator connection
	worker.POST("/:workerId/heartbeat", func(c *gin.Context) {})

	// Lease an available range
	worker.POST("/:workerId/lease", func(c *gin.Context) {})

	// Report progress metrics
	worker.POST("/:workerId/progress", func(c *gin.Context) {})

	// Notify about a completed job
	worker.POST("/:workerId/complete", func(c *gin.Context) {})

	// Notify about a failed link
	worker.POST("/:workerId/fail", func(c *gin.Context) {})

	// List all workers
	router.GET("", func(c *gin.Context) {})

	// Get one worker
	router.GET("/:workerId", func(c *gin.Context) {})

	// Patch a worker
	user.PATCH("/:workerId", func(c *gin.Context) {})

	// Delete a worker
	user.DELETE("/:workerId", func(c *gin.Context) {})

	// Get worker metrics
	router.GET("/:workerId/metrics", func(c *gin.Context) {})
}
