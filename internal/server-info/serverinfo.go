package serverinfo

import (
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/ricochet2200/go-disk-usage/du"
)

type ServerInfo interface {
	GetDiskUsage() DiskUsage
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
	nzbsPath string
}

func NewServerInfo(u usenet.UsenetConnectionPool, nzbsPath string) *serverInfo {
	return &serverInfo{u: u, nzbsPath: nzbsPath}
}

func (s *serverInfo) GetDiskUsage() DiskUsage {
	usage := du.NewDiskUsage(s.nzbsPath)

	return DiskUsage{
		Total:  usage.Size(),
		Free:   usage.Available(),
		Used:   usage.Used(),
		Folder: s.nzbsPath,
	}
}

func (s *serverInfo) GetConnections() UsenetConnections {
	return UsenetConnections{
		Total:  s.u.GetMaxConnections(),
		Free:   s.u.GetFreeConnections(),
		Active: s.u.GetActiveConnections(),
	}
}
