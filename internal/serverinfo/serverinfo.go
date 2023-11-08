package serverinfo

import (
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/ricochet2200/go-disk-usage/du"
)

type ServerInfo interface {
	GetRootFolderDiskUsage() DiskUsage
	GetConnections() UsenetConnections
	GetDownloadOnlyConnections() UsenetConnections
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
	conPool  connectionpool.UsenetConnectionPool
	rootPath string
}

func NewServerInfo(cp connectionpool.UsenetConnectionPool, rootPath string) *serverInfo {
	return &serverInfo{rootPath: rootPath, conPool: cp}
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

func (s *serverInfo) GetConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.conPool.GetMaxConnections(),
		Free:   s.conPool.GetFreeConnections(),
		Active: s.conPool.GetMaxConnections() - s.conPool.GetFreeConnections(),
	}
}

func (s *serverInfo) GetDownloadOnlyConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.conPool.GetMaxDownloadOnlyConnections(),
		Free:   s.conPool.GetDownloadOnlyFreeConnections(),
		Active: s.conPool.GetMaxDownloadOnlyConnections() - s.conPool.GetDownloadOnlyFreeConnections(),
	}
}
