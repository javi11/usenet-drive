package handlers

import (
	"net/http"
	"strconv"

	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
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
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.NoContent(http.StatusNoContent)
	}
}
