package handlers

import (
	"net/http"
	"strconv"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	echo "github.com/labstack/echo/v4"
)

func BuildGetFailedJobsHandler(queue uploadqueue.UploadQueue) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit := 10
		offset := 0
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, _ = strconv.Atoi(limitStr)
		}
		if offsetStr := c.QueryParam("offset"); offsetStr != "" {
			offset, _ = strconv.Atoi(offsetStr)
		}

		jobs, err := queue.GetFailedJobs(c.Request().Context(), limit, offset)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, jobs)
	}
}
