package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"

	"github.com/gin-gonic/gin"
)

func LoadUsersRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app), userOrAdminMiddleware(app))

	// Register
	router.POST("/register", func(c *gin.Context) {})

	// Login
	router.POST("/login", func(c *gin.Context) {})

	// Refresh
	router.POST("/refresh", func(c *gin.Context) {})

	// Get user
	auth.GET("/:userId", func(c *gin.Context) {})

	// Patch user
	auth.PATCH("/:userId", func(c *gin.Context) {})

	// Delete user
	auth.DELETE("/:userId", func(c *gin.Context) {})
}

// publicUser is the safe view of a user
func publicUser(user domain.User) gin.H {
	return gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
		"role":     user.Role,
	}
}
