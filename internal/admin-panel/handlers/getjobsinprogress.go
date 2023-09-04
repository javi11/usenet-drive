package handlers

import (
	"net/http"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/labstack/echo/v4"
)

func BuildGetJobsInProgressHandler(queue uploadqueue.UploadQueue) echo.HandlerFunc {
	return func(c echo.Context) error {
		jobs := queue.GetJobsInProgress()

		return c.JSON(http.StatusOK, jobs)
	}
}
