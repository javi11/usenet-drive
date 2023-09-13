package handlers

import (
	"net/http"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	echo "github.com/labstack/echo/v4"
)

type ManualUploadRequest struct {
	FilePath string `json:"file_path"`
}

func ManualScanHandler(queue uploadqueue.UploadQueue) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body ManualUploadRequest
		err := c.Bind(&body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if err := queue.AddJob(c.Request().Context(), body.FilePath); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.NoContent(http.StatusNoContent)
	}
}
