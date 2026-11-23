package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaronf/httpsign"
)

// Errors
var (
	ErrSignatureAuthenticationOnNonWorkerRoute = utility.NewInternalError("failed due to signature authentication on non-worker route")
	ErrInvalidUserID                           = utility.NewValidationError("validation failed due to user id not being a valid integer")
	ErrInvalidWorkerID                         = utility.NewValidationError("validation failed due to worker id not being a valid integer")
)

// bearerAuthMiddleware
func bearerAuthMiddleware(app *app.App, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		var rawToken string
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			rawToken = strings.TrimPrefix(authHeader, "Bearer ")
		} else if queryToken := c.Query("token"); queryToken != "" {
			rawToken = queryToken
		}

		if rawToken == "" {
			slog.Warn("Missing or malformed Authorization token", "method", method, "path", path)
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := app.Crypto.ParseUserToken(rawToken)
		if err != nil {
			// SendError logs the failure with request details
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			utility.SendError(c, err)
			c.Abort()
			return
		}

		if len(roles) > 0 && !slices.Contains(roles, string(claims.UserRole)) {
			slog.Warn("User role forbidden for route",
				"method", method,
				"path", path,
				"userID", claims.UserID,
				"userRole", claims.UserRole,
				"requiredRoles", roles,
			)
			utility.SendError(c, utility.NewForbiddenError("you do not have permission to perform this action (requires operator or administrator role)"))
			c.Abort()
			return
		}

		slog.Info("Authenticated request",
			"method", method,
			"path", path,
			"userID", claims.UserID,
			"userRole", claims.UserRole,
		)
		c.Set("userID", claims.UserID)
		c.Set("userRole", string(claims.UserRole))
		c.Next()
	}
}

// signatureAuthMiddleware verifies signatuer of a worker
// Signature is passed using the Signature header
// ED25519 key must be used
func signatureAuthMiddleware(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerIdS := c.Param("workerId")
		if workerIdS == "" {
			c.AbortWithError(http.StatusInternalServerError, ErrSignatureAuthenticationOnNonWorkerRoute)
			return
		}

		workerId, err := strconv.Atoi(workerIdS)
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}
		c.Set("workerID", workerId)

		pubKey, err := app.Worker.GetWorkerSignatureKeyByID(c.Request.Context(), workerId)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		method := c.Request.Method
		path := c.Request.URL.Path

		sigHeader := c.GetHeader("Signature")
		inputHeader := c.GetHeader("Signature-Input")
		if sigHeader == "" || inputHeader == "" {
			slog.Warn("Worker request is missing HTTP message signature headers",
				"method", method, "path", path, "workerID", workerId)
			c.Header("Accept-Signature",
				`sig1=("@method" "@target-uri" "content-digest" "content-type"); alg="ed25519"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		verifier, err := httpsign.NewEd25519Verifier(
			pubKey,
			httpsign.NewVerifyConfig(),
			httpsign.Headers("@method", "@target-uri", "content-digest", "content-type"),
		)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		msg, err := httpsign.NewMessage(httpsign.NewMessageConfig().WithRequest(c.Request))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		sigName := "sig1"
		if idx := strings.IndexByte(inputHeader, '='); idx > 0 {
			sigName = strings.TrimSpace(inputHeader[:idx])
		}

		if _, err := msg.Verify(sigName, *verifier); err != nil {
			slog.Warn("Worker signature verification failed",
				"method", method, "path", path, "workerID", workerId, "error", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		slog.Info("Authenticated worker request",
			"method", method,
			"path", path,
			"workerID", workerId,
		)
		c.Next()
	}
}

// userOrAdminMiddleware
// Must be used only after bearerAuthMiddleware
func userOrAdminMiddleware(_ *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("userRole")
		userID := c.GetInt("userID")

		if userRole == domain.UserRoleAdministrator {
			c.Next()
			return
		}

		pathID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}
		if userID != pathID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
