package handlers

import (
	"net/http"

	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	RootFolderDiskUsage       serverinfo.DiskUsage         `json:"root_folder_disk_usage"`
	UsenetDownloadConnections serverinfo.UsenetConnections `json:"download_usenet_connections"`
	UsenetUploadConnections   serverinfo.UsenetConnections `json:"upload_usenet_connections"`
	GlobalActivity            serverinfo.GlobalActivity    `json:"global_activity"`
}

func GetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			RootFolderDiskUsage:       si.GetRootFolderDiskUsage(),
			UsenetDownloadConnections: si.GetDownloadConnections(),
			UsenetUploadConnections:   si.GetUploadConnections(),
			GlobalActivity:            si.GetGlobalActivity(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
