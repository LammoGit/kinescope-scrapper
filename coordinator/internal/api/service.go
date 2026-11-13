package api

import (
	"coordinator/internal/app"
	"coordinator/internal/utility"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	ErrEmptyQueryParameter   = utility.NewValidationError("query parameter 'q' is required")
	ErrInvalidLimitParameter = utility.NewValidationError("query parameter 'limit' must be a positive integer")
	ErrInvalidOffsetParamter = utility.NewValidationError("query parameter 'offset' must be a non-negative integer")
	ErrInvalidVideoID        = utility.NewValidationError("validation failed due to video id not being a valid integer")
)

func LoadServiceRouter(router gin.IRouter, app *app.App) {
	router.GET("/search", func(c *gin.Context) {
		var queries struct {
			Query  string `form:"q" binding:"required"`
			Limit  int    `form:"limit" binding:"required,gt=0"`
			Offset int    `form:"offset" binding:"required,min=0"`
		}
		if err := c.ShouldBindQuery(&queries); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		videos, err := app.Video.Search(ctx, queries.Query, queries.Limit, queries.Offset)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    videos,
			"quantity": len(videos),
		})
	})

	router.GET("/recommended", func(c *gin.Context) {
		var queries struct {
			Limit  int `form:"limit" binding:"required,gt=0"`
			Offset int `form:"offset" binding:"required,min=0"`
		}
		if err := c.ShouldBindQuery(&queries); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		videos, err := app.Video.GetRecommended(ctx, queries.Limit, queries.Offset)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    videos,
			"quantity": len(videos),
		})
	})

	router.GET("/video/:videoId", func(c *gin.Context) {
		videoID, err := strconv.Atoi(c.Param("videoId"))
		if err != nil {
			utility.SendError(c, ErrInvalidVideoID)
			return
		}

		ctx := c.Request.Context()
		video, streamURL, err := app.Video.GetVideoByID(ctx, videoID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"video_id": video.ID,
			"url":      streamURL,
			"metadata": video.Metadata,
		})
	})

	router.GET("/video/:videoId/recommended", func(c *gin.Context) {
		videoID, err := strconv.Atoi(c.Param("videoId"))
		if err != nil {
			utility.SendError(c, ErrInvalidVideoID)
			return
		}

		var queries struct {
			Limit  int `form:"limit" binding:"required,gt=0"`
			Offset int `form:"offset" binding:"required,min=0"`
		}
		if err := c.ShouldBindQuery(&queries); err != nil {
			utility.SendBindingError(c, err)
			return
		}

		ctx := c.Request.Context()
		videos, err := app.Video.GetRecommendedForVideo(ctx, videoID, queries.Limit, queries.Offset)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    videos,
			"quantity": len(videos),
		})
	})
}
