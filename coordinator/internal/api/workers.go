package api

import (
	"context"
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/services"
	"coordinator/internal/utility"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidResourceID = utility.NewValidationError("validation failed due to resource id not being a valid integer")
)

func LoadWorkersRouter(router gin.IRouter, app *app.App) {
	user := router.Group("", bearerAuthMiddleware(app, domain.UserRoleAdministrator, domain.UserRoleOperator))
	worker := router.Group("", signatureAuthMiddleware(app))

	// Register a new worker
	user.POST("/register", func(c *gin.Context) {
		var payload struct {
			Name           string `json:"name" form:"name" binding:"required,min=1,max=64"`
			Host           string `json:"host" form:"host" binding:"required"`
			SSHPort        int    `json:"ssh_port" form:"ssh_port" binding:"required,port"`
			SSHUser        string `json:"ssh_user" form:"ssh_user" binding:"required"`
			SSHKey         string `json:"ssh_key" form:"ssh_key" binding:"required"`
			SSHPassword    string `json:"ssh_password" form:"ssh_password" binding:"omitempty"`
			Threads        int    `json:"threads" form:"threads" binding:"required,gt=0"`
			StoragePath    string `json:"storage_path" form:"storage_path" binding:"required,filepath"`
			MinFreeSpaceGB int    `json:"min_free_space_gb" form:"min_free_space_gb" binding:"required,min=0"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		pubKey, err := app.Provisioning.Setup(
			ctx,
			payload.Host,
			payload.SSHPort,
			payload.SSHUser,
			payload.SSHPassword,
			payload.StoragePath,
		)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		worker, err := app.Worker.CreateWorker(
			ctx,
			payload.Host,
			payload.SSHPort,
			payload.SSHUser,
			pubKey,
			payload.Threads,
			payload.StoragePath,
			payload.MinFreeSpaceGB,
		)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Header("Location", "/api/workers/"+strconv.Itoa(worker.ID))
		c.JSON(http.StatusCreated, worker)
	})

	// Heartbeat to prolong worker-coordinator connection
	worker.POST("/:workerId/heartbeat", func(c *gin.Context) {
		workerID := c.GetInt("workerID")

		var payload struct {
			Metrics *domain.Metrics `json:"metrics" form:"metrics"`
		}

		if err := c.ShouldBind(&payload); err != nil && !errors.Is(err, io.EOF) {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		if payload.Metrics != nil {
			err := app.Telemetry.AddWorkerMetricsByID(ctx, workerID, *payload.Metrics)
			if err != nil {
				utility.SendError(c, err)
				return
			}
		}

		err := app.Job.ReceiveWorkerHeartbeat(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusOK)
	})

	// Lease an available range
	worker.POST("/:workerId/lease", func(c *gin.Context) {
		workerID := c.GetInt("workerID")

		timeout := 30 * time.Second

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		r, err := app.Job.Lease(ctx, workerID)
		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				c.Status(http.StatusNoContent)
			case errors.Is(err, context.Canceled):
				c.Status(http.StatusRequestTimeout)
			default:
				utility.SendError(c, err)
			}
			return
		}

		c.JSON(http.StatusOK, r)
	})

	// Report progress metrics
	worker.POST("/:workerId/progress", func(c *gin.Context) {
		workerID := c.GetInt("workerID")

		var payload domain.Metrics
		if err := c.ShouldBindJSON(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		err := app.Telemetry.AddWorkerMetricsByID(ctx, workerID, payload)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusOK)
	})

	// Notify about a completed job
	worker.POST("/:workerId/complete", func(c *gin.Context) {
		workerID := c.GetInt("workerID")

		var payload domain.Metrics
		if err := c.ShouldBindJSON(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		err := app.Telemetry.AddWorkerMetricsByID(ctx, workerID, payload)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		if err := app.Job.WorkerCompleted(ctx, workerID, payload.RangeID); err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusOK)
	})

	// Notify about a failed link
	worker.POST("/:workerId/fail", func(c *gin.Context) {
		workerID := c.GetInt("workerID")

		var payload struct {
			URL string `json:"url" form:"url" binding:"required,url"`
			Msg string `json:"error_message" form:"error_message" binding:"required"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		err := app.Job.FailedLink(ctx, workerID, payload.URL, payload.Msg)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusOK)
	})

	// List all workers
	router.GET("", func(c *gin.Context) {
		ctx := c.Request.Context()
		workers, err := app.Worker.GetWorkers(ctx)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, workers)
	})

	// Get one worker
	router.GET("/:workerId", func(c *gin.Context) {
		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		ctx := c.Request.Context()
		worker, err := app.Worker.GetWorkerByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, worker)
	})

	// Patch a worker
	user.PATCH("/:workerId", func(c *gin.Context) {
		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		var patch map[string]any
		if err := c.ShouldBind(&patch); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		worker, err := app.Worker.GetWorkerByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		err = utility.ApplyMergePatch(worker, patch)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		err = app.Worker.ReplaceWorkerByID(ctx, worker)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, worker)
	})

	// Delete a worker
	user.DELETE("/:workerId", func(c *gin.Context) {
		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		ctx := c.Request.Context()
		err = app.Worker.DeleteWorkerByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	})

	// Get worker metrics
	router.GET("/:workerId/metrics", func(c *gin.Context) {
		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		ctx := c.Request.Context()
		metrics, err := app.Telemetry.GetWorkerMetricsByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, metrics)
	})

	// Get upload link for video download
	worker.POST("/:workerId/upload/:resourceId", func(c *gin.Context) {
		// IN DEVELOPMENT
		// Should make keys for S3 storage using VideoService, not CryptoService

		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		resourceID, err := strconv.Atoi(c.Param("resourceId"))
		if err != nil {
			utility.SendError(c, ErrInvalidResourceID)
			return
		}

		ctx := c.Request.Context()
		worker, err := app.Worker.GetWorkerByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		token, err := app.Crypto.SignUploadToken(services.Upload{
			WorkerID:     workerID,
			ResourceID:   resourceID,
			SignatureKey: worker.PublicKey,
		})
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"link":         "http://localhost:5433/upload",
			"access_token": token,
		})
	})
}
