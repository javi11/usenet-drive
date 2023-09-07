package handlers

import (
	"net/http"

	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	DiskUsage                 serverinfo.DiskUsage         `json:"disk_usage"`
	DownloadUsenetConnections serverinfo.UsenetConnections `json:"download_usenet_connections"`
}

func BuildGetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			DiskUsage:                 si.GetDiskUsage(),
			DownloadUsenetConnections: si.GetConnections(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
