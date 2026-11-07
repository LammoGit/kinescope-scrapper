package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func LoadSSHRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app, domain.UserRoleAdministrator))

	auth.GET("/:workerId", func(c *gin.Context) {})
}
