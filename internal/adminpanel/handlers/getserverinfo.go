package handlers

import (
	"net/http"

	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/labstack/echo/v4"
)

type serverInfoResponse struct {
	RootFolderDiskUsage serverinfo.DiskUsage          `json:"root_folder_disk_usage"`
	ProvidersInfo       []connectionpool.ProviderInfo `json:"providers_info"`
	GlobalActivity      serverinfo.GlobalActivity     `json:"global_activity"`
}

func GetServerInfoHandler(si serverinfo.ServerInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		result := serverInfoResponse{
			RootFolderDiskUsage: si.GetRootFolderDiskUsage(),
			ProvidersInfo:       si.GetProvidersInfo(),
			GlobalActivity:      si.GetGlobalActivity(),
		}

		return c.JSON(http.StatusOK, result)
	}
}
