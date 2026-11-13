package api

import (
	"coordinator/internal/app"
	"coordinator/pkg/sse"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func LoadStreamRouter(router gin.IRouter, app *app.App) {
	// Channel for worker to receive commands
	router.GET("/workers/commands", func(c *gin.Context) {
		var topics []string
		workerIds, ok := c.GetQueryArray("workers")
		if !ok {
			topics = []string{"worker-*"}
		} else {
			for _, id := range workerIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("worker-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) { return nil, false })
	})

	// Channel for user to receive worker updates
	router.GET("/workers/updates", func(c *gin.Context) {
		var topics []string
		workerIds, ok := c.GetQueryArray("workers")
		if !ok {
			topics = []string{"metrics-worker-*"}
		} else {
			for _, id := range workerIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("metrics-worker-%s", id))
			}
		}
		// IN DEVELOPMENT
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) { return nil, false })
	})

	// Channel for user to receive worker metrics
	router.GET("/workers/metrics", func(c *gin.Context) {
		var topics []string
		workerIds, ok := c.GetQueryArray("workers")
		if !ok {
			topics = []string{"metrics-worker-*"}
		} else {
			for _, id := range workerIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("metrics-worker-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "metrics-worker-*" {
					metrics, err := app.Telemetry.GetWorkersMetrics(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "worker.metrics", Data: metrics}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "metrics-worker-"))
				metrics, err := app.Telemetry.GetWorkerMetricsByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: "worker.metrics", Data: metrics})
			}
			return msgs, true
		})
	})

	// Channel for user to receive job updates
	router.GET("/jobs/updates", func(c *gin.Context) {
		var topics []string
		jobIds, ok := c.GetQueryArray("jobs")
		if !ok {
			topics = []string{"metrics-job-*"}
		} else {
			for _, id := range jobIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("metrics-job-%s", id))
			}
		}
		// IN DEVELOPMENT
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) { return nil, false })
	})

	// Channel for user to receive job metrics
	router.GET("/jobs/metrics", func(c *gin.Context) {
		var topics []string
		jobIds, ok := c.GetQueryArray("jobs")
		if !ok {
			topics = []string{"metrics-job-*"}
		} else {
			for _, id := range jobIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("metrics-job-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "metrics-job-*" {
					metrics, err := app.Telemetry.GetJobsMetrics(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "job.metrics", Data: metrics}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix("job.metrics", "metrics-job-"))
				metrics, err := app.Telemetry.GetJobMetricsByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: topic, Data: metrics})
			}
			return msgs, true
		})
	})

	// Channel for user to receive events
	router.GET("/events", func(c *gin.Context) {
		var topics []string
		jobIds := c.QueryArray("jobs")
		workerIds := c.QueryArray("workers")

		if len(jobIds) == 0 && len(workerIds) == 0 {
			topics = append(topics, "events-*")
		}
		for _, id := range jobIds {
			topics = append(topics, fmt.Sprintf("events-job-%s", id))
		}
		for _, id := range workerIds {
			topics = append(topics, fmt.Sprintf("events-worker-%s", id))
		}

		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "events-*" {
					events, err := app.Event.GetEvents(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "event", Data: events}}, true
				}
				if strings.HasPrefix(topic, "events-job-") {
					id, _ := strconv.Atoi(strings.TrimPrefix(topic, "events-job-"))
					events, err := app.Event.GetEventsByJobID(ctx, id)
					if err != nil {
						continue
					}
					msgs = append(msgs, sse.SSEMessage{Event: "event", Data: events})
				} else if strings.HasPrefix(topic, "events-worker-") {
					id, _ := strconv.Atoi(strings.TrimPrefix(topic, "events-worker-"))
					events, err := app.Event.GetEventsByWorkerID(ctx, id)
					if err != nil {
						continue
					}
					msgs = append(msgs, sse.SSEMessage{Event: "event", Data: events})
				}
			}
			return msgs, true
		})
	})
}
