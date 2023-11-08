package handlers

import (
	"net/http"

	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	RootFolderDiskUsage           serverinfo.DiskUsage         `json:"root_folder_disk_usage"`
	UsenetDownloadOnlyConnections serverinfo.UsenetConnections `json:"download_only_usenet_connections"`
	UsenetConnections             serverinfo.UsenetConnections `json:"usenet_connections"`
}

func GetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			RootFolderDiskUsage:           si.GetRootFolderDiskUsage(),
			UsenetDownloadOnlyConnections: si.GetDownloadOnlyConnections(),
			UsenetConnections:             si.GetConnections(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
