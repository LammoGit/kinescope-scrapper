package api

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs every completed HTTP request with its method, path, status, latency and, when available, the authenticated user
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
		}
		if raw := c.Request.URL.RawQuery; raw != "" {
			// Never log bearer tokens passed as query parameters
			q := c.Request.URL.Query()
			if _, ok := q["token"]; ok {
				q["token"] = []string{"[REDACTED]"}
			}
			attrs = append(attrs, "query", q.Encode())
		}
		if userID, ok := c.Get("userID"); ok {
			attrs = append(attrs, "userID", userID)
		}
		if workerID, ok := c.Get("workerID"); ok {
			attrs = append(attrs, "workerID", workerID)
		}

		slog.Info("Request handled", attrs...)
	}
}
