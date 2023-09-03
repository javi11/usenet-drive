package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

func BuildGetFailedJobsHandler(queue uploadqueue.UploadQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 10
		offset := 0
		if limitStr := c.Query("limit"); limitStr != "" {
			limit, _ = strconv.Atoi(limitStr)
		}
		if offsetStr := c.Query("offset"); offsetStr != "" {
			offset, _ = strconv.Atoi(offsetStr)
		}

		jobs, err := queue.GetFailedJobs(c, limit, offset)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, jobs)
	}
}
