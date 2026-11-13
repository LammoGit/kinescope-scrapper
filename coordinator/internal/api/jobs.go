package api

import (
	"context"
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// LoadJobsRouter
func LoadJobsRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app, domain.UserRoleOperator, domain.UserRoleAdministrator))

	// List all jobs
	router.GET("", func(c *gin.Context) {
		ctx := c.Request.Context()
		jobs, err := app.Job.GetJobs(ctx)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, jobs)
	})

	// Create a new job
	auth.POST("", func(c *gin.Context) {
		var payload domain.JobConfig
		if err := c.ShouldBindJSON(&payload); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		job, err := app.Job.CreateJob(ctx, "", payload)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.Header("Location", "/api/jobs/"+strconv.Itoa(job.ID))
		c.JSON(http.StatusCreated, job)
	})

	// Get a job
	router.GET("/:jobId", func(c *gin.Context) {
		jobID, ok := parseJobID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		job, err := app.Job.GetJobByID(ctx, jobID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, job)
	})

	// Patch a job
	auth.PATCH("/:jobId", func(c *gin.Context) {
		jobID, ok := parseJobID(c)
		if !ok {
			return
		}

		var patch map[string]any
		if err := c.ShouldBindJSON(&patch); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		job, err := app.Job.GetJobByID(ctx, jobID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		if err := utility.ApplyMergePatch(&job, patch); err != nil {
			utility.SendError(c, utility.NewValidationError(err.Error()))
			return
		}
		if err := binding.Validator.ValidateStruct(&job); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		err = app.Job.ReplaceJobByID(ctx, job)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, job)
	})

	// Delete a job
	auth.DELETE("/:jobId", func(c *gin.Context) {
		jobID, ok := parseJobID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		if err := app.Job.DeleteJob(ctx, jobID); err != nil {
			utility.SendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	})

	// Start a job
	auth.POST("/:jobId/start", func(c *gin.Context) {
		changeJobState(c, app.Job.StartJob)
	})

	// Pause a job
	auth.POST("/:jobId/pause", func(c *gin.Context) {
		changeJobState(c, app.Job.PauseJob)
	})

	// Resume a job
	auth.POST("/:jobId/resume", func(c *gin.Context) {
		changeJobState(c, app.Job.ResumeJob)
	})

	// Stop a job
	auth.POST("/:jobId/stop", func(c *gin.Context) {
		changeJobState(c, app.Job.StopJob)
	})

	// List identifiers of videos downloaded for a job
	router.GET("/:jobId/items", func(c *gin.Context) {
		jobID, ok := parseJobID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		if _, err := app.Job.GetJobByID(ctx, jobID); err != nil {
			utility.SendError(c, err)
			return
		}

		videos, err := app.Video.GetVideosByJobID(ctx, jobID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		items := make([]int, 0, len(videos))
		for _, video := range videos {
			items = append(items, video.ID)
		}

		c.JSON(http.StatusOK, items)
	})

	// List events for a job
	router.GET("/:jobId/events", func(c *gin.Context) {
		jobID, ok := parseJobID(c)
		if !ok {
			return
		}

		ctx := c.Request.Context()
		if _, err := app.Job.GetJobByID(ctx, jobID); err != nil {
			utility.SendError(c, err)
			return
		}

		events, err := app.Event.GetEventsByJobID(ctx, jobID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, events)
	})
}

// parseJobID parses the jobId path parameter,
// sending an error response when it isn't a valid integer
func parseJobID(c *gin.Context) (int, bool) {
	jobID, err := strconv.Atoi(c.Param("jobId"))
	if err != nil {
		utility.SendError(c, utility.NewValidationError("Validation failed due to jobId not being a valid integer"))
		return 0, false
	}
	return jobID, true
}

// changeJobState applies a lifecycle operation and writes the updated job
func changeJobState(c *gin.Context, change func(ctx context.Context, id int) error) {
	jobID, ok := parseJobID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	err := change(ctx, jobID)
	if err != nil {
		utility.SendError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
