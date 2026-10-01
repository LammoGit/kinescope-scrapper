package api

import (
	"bytes"
	"coordinator/internal/api/stream"
	st "coordinator/internal/storage"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func base64Decode(s string) ([]byte, error) {
	sig, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		sig, err = base64.RawStdEncoding.DecodeString(s)
	}
	return sig, err
}

func noopHandler(c *gin.Context) {}

func authMiddleware(storage *st.Storage, register ...bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()

		signatureString := c.Request.Header.Get("X-Signature")
		timestamp := c.Request.Header.Get("X-Timestamp")

		var prefix string
		if len(register) > 0 && register[0] {
			prefix = fmt.Sprintf("%s:", timestamp)
		} else {
			workerID := c.Param("workerId")
			prefix = fmt.Sprintf("%s:%s:", timestamp, workerID)
		}

		switch true {
		case signatureString == "":
			c.String(http.StatusUnauthorized, "The X-Signature header's value must be a base64 encoded ed25519 signature")
			c.Abort()
			return
		case timestamp == "":
			c.String(http.StatusUnauthorized, "The value of the X-Timestamp header must be a number of nanoseconds of type integer")
			c.Abort()
			return
		}

		nsec, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "The value of the X-Timestamp header must be a number of nanoseconds of type integer")
			c.Abort()
			return
		}

		expirationTime := time.Unix(0, nsec).Add(5 * time.Minute)
		if now.After(expirationTime) {
			c.String(http.StatusUnauthorized, "The X-Timestamp header is expired")
			c.Abort()
			return
		}

		var buf bytes.Buffer
		if c.Request.Body != nil {
			_, _ = io.Copy(&buf, c.Request.Body)
		}
		rawBodyBytes := buf.Bytes()

		data := make(map[string]any)
		err = json.Unmarshal(rawBodyBytes, &data)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		var pub ed25519.PublicKey
		if len(register) > 0 && register[0] {
			pubBase64, ok := data["public_key"].(string)
			if !ok {
				c.String(http.StatusBadRequest, "Request's body doesn't have public key")
				c.Abort()
				return
			}
			pub, err = base64Decode(pubBase64)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			workerID := c.Param("workerId")
			server, ok := storage.Servers[workerID]
			if !ok {
				c.String(http.StatusBadRequest, "Worker with this id doesn't exist")
				c.Abort()
				return
			}
			pub = server.Info.PublicKey
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBodyBytes))

		sig, err := base64.StdEncoding.DecodeString(signatureString)
		if err != nil {
			sig, err = base64.RawStdEncoding.DecodeString(signatureString)
			if err != nil {
				c.String(http.StatusBadRequest, "The X-Signature header's value must be a valid base64 encoded ed25519 signature")
				c.Abort()
				return
			}
		}

		signedBytes := append([]byte(prefix), rawBodyBytes...)

		if !ed25519.Verify(pub, signedBytes, sig) {
			c.String(http.StatusUnauthorized, "Verification failed due to signature not matching public key")
			c.Abort()
			return
		}

		c.Next()
	}
}

func LoadRouter(router gin.IRouter, hub *stream.Hub, storage *st.Storage) {
	publishWorkerUpdate := func(workerID, event string) {
		info, _ := storage.GetWorkerInfo(workerID)
		metrics, _ := storage.GetWorkerMetrics(workerID)

		hub.PublishAll(
			[]string{stream.TopicWorkers, stream.WorkerTopic(workerID)},
			stream.SSEMessage{
				Event: event,
				Data: gin.H{
					"worker_id": workerID,
					"info":      info,
					"metrics":   metrics,
				},
			},
		)
	}

	// --- `/jobs/*` ---
	jobs := router.Group("/jobs")

	// Creates a new job
	jobs.POST("/", func(c *gin.Context) {
		var buf bytes.Buffer
		io.Copy(&buf, c.Request.Body)

		var job st.Job
		if err := json.Unmarshal(buf.Bytes(), &job); err != nil {
			c.String(http.StatusBadRequest, "Failed to unmarshal received JSON body: %s", err)
			return
		}
		jobID := storage.CreateJob(job.From, job.To)
		c.String(http.StatusOK, jobID)
	})

	// Sends a map with jobId as key and job info as value
	jobs.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, storage.GetJobs())
	})

	// Sends job info by jobId
	jobs.GET("/:jobId", func(c *gin.Context) {
		jobId := c.Param("jobId")
		job, ok := storage.GetJob(jobId)
		if !ok {
			c.String(http.StatusNotFound, "")
			return
		}
		c.JSON(http.StatusOK, job)
	})

	// Starts a job by jobId
	jobs.POST("/:jobId/start", func(c *gin.Context) {
		jobId := c.Param("jobId")
		storage.StartJob(jobId)
		c.Status(http.StatusOK)
	})

	// Pauses a job by jobId
	jobs.POST("/:jobId/pause", func(c *gin.Context) {
		jobId := c.Param("jobId")
		storage.PauseJob(jobId)
		c.Status(http.StatusOK)
	})

	// Resumes paused a job by jobId
	jobs.POST("/:jobId/resume", func(c *gin.Context) {
		jobId := c.Param("jobId")
		storage.ResumeJob(jobId)
		c.Status(http.StatusOK)
	})

	// Stops a job by jobId
	jobs.POST("/:jobId/stop", func(c *gin.Context) {
		jobId := c.Param("jobId")
		storage.StopJob(jobId)
		c.Status(http.StatusOK)
	})

	// Sends a list of downloaded videos
	jobs.GET("/:jobId/items", func(c *gin.Context) {
		jobId := c.Param("jobId")
		c.Status(http.StatusOK, storage.JobItems(jobId))
	})

	// Sends a list of events
	jobs.GET("/:jobId/events", func(c *gin.Context) {
		jobId := c.Param("jobId")
		c.Status(http.StatusOK, storage.JobEvents(jobId))
	})

	// --- `/workers/*` ---
	workers := router.Group("/workers")

	// Add new worker to the coordinator config
	workers.POST("/register",
		authMiddleware(storage, true),
		func(c *gin.Context) {
			var buf bytes.Buffer
			io.Copy(&buf, c.Request.Body)

			var serverInfo st.ServerInfo
			if err := json.Unmarshal(buf.Bytes(), &serverInfo); err != nil {
				c.String(http.StatusBadRequest, "Failed to unmarshal received JSON body: %s", err)
				return
			}

			workerID, sshKey, err := storage.RegisterServer(serverInfo)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to register: %s", err)
				return
			}

			publishWorkerUpdate(workerID, "worker_registered")

			c.JSON(http.StatusOK, gin.H{
				"worker_id": workerID,
				"ssh_key":   sshKey,
			})
		},
	)

	// Receive heartbeat to know if server alive
	workers.POST("/:workerId/heartbeat",
		authMiddleware(storage),
		func(c *gin.Context) {
			// Update status info for the worker
			c.Status(http.StatusOK)
		},
	)

	// Gives out job to a worker on request
	workers.POST("/:workerId/lease",
		authMiddleware(storage),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"job_id": "job-01",
				"name":   "job-01",
				"from":   0,
				"to":     1000,
			})
		},
	)

	// Receives and stores progress data
	workers.POST("/:workerId/progress",
		authMiddleware(storage),
		func(c *gin.Context) {
			var buf bytes.Buffer
			io.Copy(&buf, c.Request.Body)

			var metrics st.ServerMetrics
			if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
				c.Status(http.StatusBadRequest)
				return
			}

			workerID := c.Param("workerId")
			storage.UpdateServerMetrics(workerID, metrics)

			publishWorkerUpdate(workerID, "worker_metrics")

			c.Status(http.StatusOK)
		},
	)

	// Finishes job for a worker once received
	workers.POST("/:workerId/complete",
		authMiddleware(storage),
		func(c *gin.Context) {
			var buf bytes.Buffer
			io.Copy(&buf, c.Request.Body)

			var metrics st.ServerMetrics
			if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
				c.Status(http.StatusBadRequest)
				return
			}

			workerID := c.Param("workerId")
			storage.UpdateServerMetrics(workerID, metrics)

			publishWorkerUpdate(workerID, "worker_status")

			c.Status(http.StatusOK)
		},
	)

	// Don't know
	workers.POST("/:workerId/fail",
		authMiddleware(storage),
		func(c *gin.Context) {
			var buf bytes.Buffer
			io.Copy(&buf, c.Request.Body)

			var metrics st.ServerMetrics
			if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
				c.Status(http.StatusBadRequest)
				return
			}

			workerID := c.Param("workerId")
			storage.UpdateServerMetrics(workerID, metrics)

			publishWorkerUpdate(workerID, "worker_status")

			c.Status(http.StatusOK)
		},
	)

	// Sends a map with workerId as key and config info as value
	workers.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, storage.GetWorkers())
	})

	// Sends config info of a worker by it's id
	workers.GET("/:workerId", func(c *gin.Context) {
		workerID := c.Param("workerId")
		info, ok := storage.GetWorkerInfo(workerID)
		if !ok {
			c.String(http.StatusNotFound, "Worker doesn't exist")
			return
		}
		c.JSON(http.StatusOK, info)
	})

	// Updates server config info by sent JSON body and it's workerId
	workers.PATCH("/:workerId", func(c *gin.Context) {
		var buf bytes.Buffer
		io.Copy(&buf, c.Request.Body)

		var info st.ServerInfo
		if err := json.Unmarshal(buf.Bytes(), &info); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// storage.UpdateServerInfo(c.Param("workerId"), info)
		// publishWorkerUpdate(c.Param("workerId"), "worker_updated")

		c.Status(http.StatusOK)
	})

	// Sends metrics of a worker by it's id
	workers.GET("/:workerId/metrics", func(c *gin.Context) {
		workerID := c.Param("workerId")
		metrics, ok := storage.GetWorkerMetrics(workerID)
		if !ok {
			c.String(http.StatusNotFound, "Worker doesn't exist")
			return
		}
		c.JSON(http.StatusOK, metrics)
	})

	// `/stream/*`
	stream.LoadRouter(router.Group("/stream"), hub, storage)
}
