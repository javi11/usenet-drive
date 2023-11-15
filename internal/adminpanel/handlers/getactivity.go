package handlers

import (
	"net/http"

	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/labstack/echo/v4"
)

func GetActivityHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, si.GetActivity())
	}
}
