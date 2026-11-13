package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
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
			topics = []string{"worker-update-*"}
		} else {
			for _, id := range workerIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("worker-update-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "worker-update-*" {
					workers, err := app.Worker.GetWorkers(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "worker.update", Data: workers}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "worker-update-"))
				worker, err := app.Worker.GetWorkerByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: "worker.update", Data: []domain.Worker{worker}})
			}
			return msgs, true
		})
	})

	// Channel for user to receive worker metrics
	router.GET("/workers/metrics", func(c *gin.Context) {
		var topics []string
		workerIds, ok := c.GetQueryArray("workers")
		if !ok {
			topics = []string{"worker-metrics-*"}
		} else {
			for _, id := range workerIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("worker-metrics-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "worker-metrics-*" {
					metrics, err := app.Telemetry.GetWorkersMetrics(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "worker.metrics", Data: metrics}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "worker-metrics-"))
				metrics, err := app.Telemetry.GetWorkerMetricsByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: "worker.metrics", Data: []domain.Metrics{metrics}})
			}
			return msgs, true
		})
	})

	// Channel for user to receive job updates
	router.GET("/jobs/updates", func(c *gin.Context) {
		var topics []string
		jobIds, ok := c.GetQueryArray("jobs")
		if !ok {
			topics = []string{"job-metrics-*"}
		} else {
			for _, id := range jobIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("job-metrics-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "job-update-*" {
					jobs, err := app.Job.GetJobs(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "job.update", Data: jobs}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "job-update-"))
				job, err := app.Job.GetJobByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: "job.update", Data: []domain.Job{job}})
			}
			return msgs, true
		})
	})

	// Channel for user to receive job metrics
	router.GET("/jobs/metrics", func(c *gin.Context) {
		var topics []string
		jobIds, ok := c.GetQueryArray("jobs")
		if !ok {
			topics = []string{"job-metrics-*"}
		} else {
			for _, id := range jobIds {
				if _, err := strconv.Atoi(id); err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				topics = append(topics, fmt.Sprintf("job-metrics-%s", id))
			}
		}
		app.Hub.ServeSSE(c, topics, func() ([]sse.SSEMessage, bool) {
			ctx := c.Request.Context()
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "job-metrics-*" {
					metrics, err := app.Telemetry.GetJobsMetrics(ctx)
					if err != nil {
						return nil, false
					}
					return []sse.SSEMessage{{Event: "job.metrics", Data: metrics}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix("job.metrics", "job-metrics-"))
				metrics, err := app.Telemetry.GetJobMetricsByID(ctx, id)
				if err != nil {
					continue
				}
				msgs = append(msgs, sse.SSEMessage{Event: topic, Data: []domain.Metrics{metrics}})
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
				if ids, ok := strings.CutPrefix(topic, "events-job-"); ok {
					id, _ := strconv.Atoi(ids)
					events, err := app.Event.GetEventsByJobID(ctx, id)
					if err != nil {
						continue
					}
					msgs = append(msgs, sse.SSEMessage{Event: "event", Data: events})
				} else if ids, ok := strings.CutPrefix(topic, "events-worker-"); ok {
					id, _ := strconv.Atoi(ids)
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
