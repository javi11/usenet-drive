package handlers

import (
	"net/http"

	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	RootFolderDiskUsage       serverinfo.DiskUsage         `json:"root_folder_disk_usage"`
	TmpFolderDiskUsage        serverinfo.DiskUsage         `json:"tmp_folder_disk_usage"`
	DownloadUsenetConnections serverinfo.UsenetConnections `json:"download_usenet_connections"`
}

func BuildGetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			RootFolderDiskUsage:       si.GetRootFolderDiskUsage(),
			TmpFolderDiskUsage:        si.GetTmpFolderDiskUsage(),
			DownloadUsenetConnections: si.GetConnections(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
