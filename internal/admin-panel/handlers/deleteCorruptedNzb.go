package handlers

import (
	"net/http"
	"strconv"

	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/corrupted-nzbs-manager"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	echo "github.com/labstack/echo/v4"
)

func DeleteCorruptedNzbHandler(cNzb corruptednzbsmanager.CorruptedNzbsManager) echo.HandlerFunc {
	return func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if err := cNzb.Delete(c.Request().Context(), id); err != nil {
			if err == uploadqueue.ErrJobNotFound {
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.NoContent(http.StatusNoContent)
	}
}
