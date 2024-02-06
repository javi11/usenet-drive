package serverinfo

import (
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	status "github.com/javi11/usenet-drive/internal/usenet/statusreporter"
	"github.com/ricochet2200/go-disk-usage/du"
)

type ServerInfo interface {
	GetRootFolderDiskUsage() DiskUsage
	GetProvidersInfo() []connectionpool.ProviderInfo
	GetActivity() []Activity
	GetGlobalActivity() GlobalActivity
}

type GlobalActivity struct {
	DownloadSpeed float64 `json:"download_speed"`
	UploadSpeed   float64 `json:"upload_speed"`
}

type Activity struct {
	SessionId    string      `json:"session_id"`
	Path         string      `json:"path"`
	CurrentSpeed float64     `json:"speed"`
	TotalBytes   int64       `json:"total_bytes"`
	Kind         status.Kind `json:"kind"`
}

type DiskUsage struct {
	Total  uint64 `json:"total"`
	Free   uint64 `json:"free"`
	Used   uint64 `json:"used"`
	Folder string `json:"folder"`
}

type serverInfo struct {
	conPool  connectionpool.UsenetConnectionPool
	rootPath string
	sr       status.StatusReporter
}

func NewServerInfo(cp connectionpool.UsenetConnectionPool, sr status.StatusReporter, rootPath string) ServerInfo {
	return &serverInfo{rootPath: rootPath, conPool: cp, sr: sr}
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

func (s *serverInfo) GetProvidersInfo() []connectionpool.ProviderInfo {
	return s.conPool.GetProvidersInfo()
}

func (s *serverInfo) GetActivity() []Activity {
	activity := make([]Activity, 0)

	for id, s := range s.sr.GetStatus() {
		activity = append(activity, Activity{
			Path:         s.Path,
			CurrentSpeed: s.CurrentSpeed,
			TotalBytes:   s.TotalBytes,
			Kind:         s.Kind,
			SessionId:    id.String(),
		})
	}

	return activity
}

func (s *serverInfo) GetGlobalActivity() GlobalActivity {
	activity := s.GetActivity()

	var downloadSpeed float64
	var uploadSpeed float64

	for _, a := range activity {
		if a.Kind == status.Download {
			downloadSpeed += a.CurrentSpeed
		} else {
			uploadSpeed += a.CurrentSpeed
		}
	}

	return GlobalActivity{
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
	}
}
