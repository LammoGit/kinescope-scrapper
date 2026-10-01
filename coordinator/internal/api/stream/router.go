package stream

import (
	st "coordinator/internal/storage"

	"github.com/gin-gonic/gin"
)

func LoadRouter(router gin.IRouter, hub *Hub, storage *st.Storage) {
	router.GET("/workers", func(c *gin.Context) {
		hub.ServeSSE(c, TopicWorkers, func() SSEMessage {
			return SSEMessage{Event: "snapshot", Data: storage.GetWorkers()}
		})
	})

	router.GET("/workers/:workerId", func(c *gin.Context) {
		id := c.Param("workerId")
		hub.ServeSSE(c, WorkerTopic(id), func() SSEMessage {
			info, ok := storage.GetWorkerInfo(id)
			if !ok {
				return SSEMessage{Event: "error", Data: gin.H{"error": "worker not found"}}
			}
			metrics, _ := storage.GetWorkerMetrics(id)
			return SSEMessage{
				Event: "snapshot",
				Data:  gin.H{"worker_id": id, "info": info, "metrics": metrics},
			}
		})
	})

	router.GET("/workers/:workerId/metrics", func(c *gin.Context) {
		id := c.Param("workerId")
		hub.ServeSSE(c, WorkerTopic(id), func() SSEMessage {
			metrics, ok := storage.GetWorkerMetrics(id)
			if !ok {
				return SSEMessage{Event: "error", Data: gin.H{"error": "worker not found"}}
			}
			return SSEMessage{Event: "snapshot", Data: metrics}
		})
	})

	router.GET("/jobs", func(c *gin.Context) {
		hub.ServeSSE(c, TopicJobs, func() SSEMessage {
			jobs := storage.GetJobs() // map[string]Job
			list := make([]gin.H, 0, len(jobs))
			for id, job := range jobs {
				list = append(list, gin.H{
					"id":         id,
					"name":       job.Name,
					"range_from": job.From,
					"range_to":   job.To,
					"status":     "working",
				})
			}
			return SSEMessage{Event: "snapshot", Data: list}
		})
	})

	router.GET("/jobs/:jobId", func(c *gin.Context) {
		id := c.Param("jobId")
		hub.ServeSSE(c, JobTopic(id), func() SSEMessage {
			job, ok := storage.GetJob(id)
			if !ok {
				return SSEMessage{Event: "error", Data: gin.H{"error": "job not found"}}
			}
			return SSEMessage{Event: "snapshot", Data: job}
		})
	})

	router.GET("/jobs/:jobId/events", func(c *gin.Context) {
		id := c.Param("jobId")
		hub.ServeSSE(c, JobEventsTopic(id), func() SSEMessage {
			return SSEMessage{Event: "snapshot", Data: []any{}}
		})
	})
}
