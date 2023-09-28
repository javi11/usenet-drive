package serverinfo

import (
	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	"github.com/ricochet2200/go-disk-usage/du"
)

type ServerInfo interface {
	GetRootFolderDiskUsage() DiskUsage
	GetDownloadConnections() UsenetConnections
	GetUploadConnections() UsenetConnections
}

type DiskUsage struct {
	Total  uint64 `json:"total"`
	Free   uint64 `json:"free"`
	Used   uint64 `json:"used"`
	Folder string `json:"folder"`
}

type UsenetConnections struct {
	Total  int `json:"total"`
	Free   int `json:"free"`
	Active int `json:"active"`
}

type serverInfo struct {
	downloadCp connectionpool.UsenetConnectionPool
	uploadCp   connectionpool.UsenetConnectionPool
	rootPath   string
}

func NewServerInfo(dlCp connectionpool.UsenetConnectionPool, upCp connectionpool.UsenetConnectionPool, rootPath string) *serverInfo {
	return &serverInfo{rootPath: rootPath, downloadCp: dlCp, uploadCp: upCp}
}

func (s *serverInfo) GetRootFolderDiskUsage() DiskUsage {
	usage := du.NewDiskUsage(s.rootPath)

	return DiskUsage{
		Total:  usage.Size(),
		Free:   usage.Available(),
		Used:   usage.Used(),
		Folder: s.rootPath,
	}
}

func (s *serverInfo) GetDownloadConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.downloadCp.GetMaxConnections(),
		Free:   s.downloadCp.GetFreeConnections(),
		Active: s.downloadCp.GetActiveConnections(),
	}
}

func (s *serverInfo) GetUploadConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.uploadCp.GetMaxConnections(),
		Free:   s.uploadCp.GetFreeConnections(),
		Active: s.uploadCp.GetActiveConnections(),
	}
}
