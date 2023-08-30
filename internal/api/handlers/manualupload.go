package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

type ManualUploadRequest struct {
	FilePath string `json:"file_path"`
}

func BuildManualUploadHandler(queue uploadqueue.UploadQueue) gin.HandlerFunc {
	return func(c *gin.Context) {

		var body ManualUploadRequest
		err := c.ShouldBind(&body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := queue.AddJob(c, body.FilePath); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
