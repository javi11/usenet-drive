package handlers

import (
	"net/http"
	"strconv"

	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
	echo "github.com/labstack/echo/v4"
)

func GetCorruptedNzbListHandler(cNzb corruptednzbsmanager.CorruptedNzbsManager) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit := 10
		offset := 0
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, _ = strconv.Atoi(limitStr)
		}
		if offsetStr := c.QueryParam("offset"); offsetStr != "" {
			offset, _ = strconv.Atoi(offsetStr)
		}

		result, err := cNzb.List(c.Request().Context(), limit, offset)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, result)
	}
}
