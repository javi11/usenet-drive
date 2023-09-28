package handlers

import (
	"net/http"

	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	RootFolderDiskUsage       serverinfo.DiskUsage         `json:"root_folder_disk_usage"`
	UploadUsenetConnections   serverinfo.UsenetConnections `json:"upload_usenet_connections"`
	DownloadUsenetConnections serverinfo.UsenetConnections `json:"download_usenet_connections"`
}

func GetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			RootFolderDiskUsage:       si.GetRootFolderDiskUsage(),
			UploadUsenetConnections:   si.GetUploadConnections(),
			DownloadUsenetConnections: si.GetDownloadConnections(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
