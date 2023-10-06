package handlers

import (
	"net/http"

	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	echo "github.com/labstack/echo/v4"
)

type deleteCorruptedNzbRequest struct {
	Path string `json:"path"`
}

func DeleteCorruptedNzbHandler(cNzb corruptednzbsmanager.CorruptedNzbsManager) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := new(deleteCorruptedNzbRequest)
		if err := c.Bind(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if err := cNzb.Delete(c.Request().Context(), req.Path); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.NoContent(http.StatusNoContent)
	}
}
