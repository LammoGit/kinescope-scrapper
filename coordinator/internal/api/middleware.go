package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaronf/httpsign"
)

// Errors
var (
	ErrSignatureAuthenticationOnNonWorkerRoute = utility.NewInternalError("Failed due to signature authentication on non-worker route")
	ErrInvalidUserID                           = utility.NewValidationError("Validation failed due to userId not being a valid integer")
	ErrNonIntegerWorkerId                      = utility.NewValidationError("Validation failed due to workerId not being a valid integer")
)

// bearerAuthMiddleware
func bearerAuthMiddleware(app *app.App, roles ...domain.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := app.Crypto.ParseToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			utility.SendError(c, err)
			return
		}
		if len(roles) > 0 {
			for _, r := range roles {
				if r == claims.UserRole {
					c.Set("userID", claims.UserID)
					c.Set("userRole", claims.UserRole)
					c.Next()
					return
				}
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.UserRole)
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
			utility.SendError(c, ErrNonIntegerWorkerId)
			return
		}

		pubKey, err := app.Worker.GetWorkerSignatureKeyByID(workerId)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		sigHeader := c.GetHeader("Signature")
		inputHeader := c.GetHeader("Signature-Input")
		if sigHeader == "" || inputHeader == "" {
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
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// userOrAdminMiddleware
func userOrAdminMiddleware(app *app.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, err := app.Crypto.ParseToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			utility.SendError(c, err)
			return
		}

		if claims.UserRole == domain.UserRoleAdministrator {
			c.Next()
			return
		}

		pathID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			utility.SendError(c, ErrInvalidUserID)
			return
		}
		if claims.UserID != pathID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
