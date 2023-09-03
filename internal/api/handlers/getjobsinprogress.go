package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

func BuildGetJobsInProgressHandler(queue uploadqueue.UploadQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobs := queue.GetJobsInProgress()

		c.JSON(http.StatusOK, jobs)
	}
}
