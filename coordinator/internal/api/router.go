package api

import (
	"coordinator/internal/app"

	"github.com/gin-gonic/gin"
)

// LoadRouter loads API routes into a given router
func LoadRouter(router gin.IRouter, app *app.App) {
	LoadJobsRouter(router.Group("/jobs"), app)
	LoadWorkersRouter(router.Group("/workers"), app)
	LoadSSHRouter(router.Group("/ssh"), app)
	LoadStreamRouter(router.Group("/stream"), app)
	LoadUsersRouter(router.Group("/users"), app)
	LoadServiceRouter(router.Group("/service"), app)
}
