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
	router.GET("/workers", func(c *gin.Context) {
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

	router.GET("/metrics/workers", func(c *gin.Context) {
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
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "metrics-worker-*" {
					return []sse.SSEMessage{{Event: topic, Data: app.Telemetry.GetWorkersMetrics()}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "metrics-worker-"))
				metrics, _ := app.Telemetry.GetWorkerMetricsByID(id)
				msgs = append(msgs, sse.SSEMessage{Event: topic, Data: metrics})
			}
			return msgs, true
		})
	})

	router.GET("/metrics/jobs", func(c *gin.Context) {
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
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "metrics-job-*" {
					return []sse.SSEMessage{{Event: topic, Data: app.Telemetry.GetJobsMetrics()}}, true
				}
				id, _ := strconv.Atoi(strings.TrimPrefix(topic, "metrics-job-"))
				metrics, _ := app.Telemetry.GetJobMetricsByID(id)
				msgs = append(msgs, sse.SSEMessage{Event: topic, Data: metrics})
			}
			return msgs, true
		})
	})

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
			var msgs []sse.SSEMessage
			for _, topic := range topics {
				if topic == "events-*" {
					return []sse.SSEMessage{{Event: topic, Data: app.Event.GetEvents()}}, true
				}
				if strings.HasPrefix(topic, "events-job-") {
					id, _ := strconv.Atoi(strings.TrimPrefix(topic, "events-job-"))
					events, _ := app.Event.GetEventsByJobID(id)
					msgs = append(msgs, sse.SSEMessage{Event: topic, Data: events})
				} else if strings.HasPrefix(topic, "events-worker-") {
					id, _ := strconv.Atoi(strings.TrimPrefix(topic, "events-worker-"))
					events, _ := app.Event.GetEventsByWorkerID(id)
					msgs = append(msgs, sse.SSEMessage{Event: topic, Data: events})
				}
			}
			return msgs, true
		})
	})
}
