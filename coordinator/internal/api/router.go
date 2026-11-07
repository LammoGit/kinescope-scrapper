package api

import (
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"

	"github.com/gin-gonic/gin"
)

// LoadRouter loads API routes into a given router.
func LoadRouter(router gin.IRouter, hub *sse.Hub, storage st.Storage) {
	LoadJobsRouter(router.Group("/jobs"), hub, storage)
	LoadWorkersRouter(router.Group("/workers"), hub, storage)
	LoadSSHRouter(router.Group("/ssh"), hub, storage)
	LoadStreamRouter(router.Group("/stream"), hub, storage)
	LoadUsersRouter(router.Group("/users"), hub, storage)
	LoadServiceRouter(router.Group("/service"), hub, storage)
}
