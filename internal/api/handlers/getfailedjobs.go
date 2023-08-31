package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

func BuildGetFailedJobsHandler(queue uploadqueue.UploadQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobs, err := queue.GetFailedJobs(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, jobs)
	}
}
