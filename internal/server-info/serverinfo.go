package serverinfo

import (
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/ricochet2200/go-disk-usage/du"
)

type ServerInfo interface {
	GetRootFolderDiskUsage() DiskUsage
	GetTmpFolderDiskUsage() DiskUsage
	GetConnections() UsenetConnections
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
	u        usenet.UsenetConnectionPool
	rootPath string
	tmpPath  string
}

func NewServerInfo(u usenet.UsenetConnectionPool, rootPath, tmpPath string) *serverInfo {
	return &serverInfo{u: u, rootPath: rootPath, tmpPath: tmpPath}
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

func (s *serverInfo) GetTmpFolderDiskUsage() DiskUsage {
	usage := du.NewDiskUsage(s.tmpPath)

	return DiskUsage{
		Total:  usage.Size(),
		Free:   usage.Available(),
		Used:   usage.Used(),
		Folder: s.tmpPath,
	}
}

func (s *serverInfo) GetConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.u.GetMaxConnections(),
		Free:   s.u.GetFreeConnections(),
		Active: s.u.GetActiveConnections(),
	}
}
