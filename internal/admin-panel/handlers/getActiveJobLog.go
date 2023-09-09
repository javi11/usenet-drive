package handlers

import (
	"net/http"
	"strconv"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/labstack/echo/v4"
)

func BuildGetActiveJobLogHandler(queue uploadqueue.UploadQueue) echo.HandlerFunc {
	return func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		log, err := queue.GetActiveJobLog(id)
		if err != nil {
			if err == uploadqueue.ErrJobNotFound {
				return c.String(http.StatusOK, "")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.String(http.StatusOK, log)
	}
}
