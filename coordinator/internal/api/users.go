package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Errors
var (
	ErrInvalidUserPassword = utility.NewConflictError("password is invalid")
	ErrRefreshExpected     = utility.NewValidationError("refresh token expected")
)

// LoadUsersRouter
func LoadUsersRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app), userOrAdminMiddleware(app))
	admin := router.Group("", bearerAuthMiddleware(app, domain.UserRoleAdministrator))

	// Register a new user
	router.POST("/register", func(c *gin.Context) {
		var payload struct {
			Username string `json:"username" form:"username" binding:"required,min=3,alphanumunicode"`
			Email    string `json:"email" form:"email" binding:"required,email"`
			Password string `json:"password" form:"password" binding:"required,min=8,alphanum"`
		}
		if err := c.ShouldBind(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		salt, hash, err := app.Crypto.HashPassword(payload.Password)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		user, err := app.User.Register(c.Request.Context(), payload.Username, payload.Email, hash, salt)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusCreated, publicUser(user))
	})

	// Login as a user
	router.POST("/login", func(c *gin.Context) {
		var payload struct {
			Email    string `json:"email" form:"email" binding:"required,email"`
			Password string `json:"password" form:"password" binding:"required,min=8,alphanum"`
		}
		if err := c.ShouldBind(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		user, err := app.User.GetUserByEmail(c.Request.Context(), payload.Email)
		if err != nil {
			utility.SendError(c, err)
			return
		}
		if !app.Crypto.VerifyPassword(payload.Password, user.Password) {
			utility.SendError(c, ErrInvalidUserPassword)
			return
		}

		access, refresh, err := app.Crypto.SignUserTokens(user)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
	})

	// Refresh user's JWT tokens
	router.POST("/refresh", func(c *gin.Context) {
		var payload struct {
			RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
		}
		if err := c.ShouldBind(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		claims, err := app.Crypto.ParseUserToken(payload.RefreshToken)
		if err != nil {
			utility.SendError(c, err)
			return
		}
		if claims.Type != "refresh" {
			utility.SendError(c, ErrRefreshExpected)
			return
		}

		user, err := app.User.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		access, refresh, err := app.Crypto.SignUserTokens(user)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
	})

	// Get user
	auth.GET("/:userId", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}

		user, err := app.User.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, publicUser(user))
	})

	// Patch user
	auth.PATCH("/:userId", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}

		var patch map[string]any
		if err := c.ShouldBindJSON(&patch); err != nil {
			utility.SendBindingError(c, err)
			return
		}
		for _, f := range []string{"password", "user_role"} {
			if _, ok := patch[f]; ok {
				utility.SendError(c, utility.NewValidationError(f+" must be changed via its own endpoint"))
				return
			}
		}

		ctx := c.Request.Context()
		user, err := app.User.GetUserByID(ctx, userID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		if err := utility.ApplyMergePatch(&user, patch); err != nil {
			utility.SendError(c, utility.NewValidationError(err.Error()))
			return
		}
		if err := binding.Validator.ValidateStruct(&user); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		err = app.User.ReplaceUser(ctx, user)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, publicUser(user))
	})

	// Change password
	auth.POST("/:userId/password", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}

		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password" binding:"required,min=8,alphanum"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		user, err := app.User.GetUserByID(ctx, userID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		isAdmin := c.GetString("userRole") == domain.UserRoleAdministrator
		if !isAdmin && !app.Crypto.VerifyPassword(body.CurrentPassword, user.Password) {
			utility.SendError(c, ErrInvalidUserPassword)
			return
		}

		salt, hash, err := app.Crypto.HashPassword(body.NewPassword)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		user.Password = domain.PasswordCredentials{Hash: hash, Salt: salt}
		if err := app.User.ReplaceUser(ctx, user); err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	})

	// Change role
	admin.PATCH("/:userId/role", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}

		var body struct {
			Role string `json:"user_role" binding:"required,oneof=viewer operator administrator"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		user, err := app.User.GetUserByID(ctx, userID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		user.Role = body.Role
		if err := app.User.ReplaceUser(ctx, user); err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, publicUser(user))
	})

	// Delete user
	auth.DELETE("/:userId", func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}

		if err := app.User.DeleteByID(c.Request.Context(), userID); err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	})
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
