package webdav

import (
	"io/fs"

	"golang.org/x/net/webdav"
)

type RemoteFileWriter interface {
	OpenFile(name string, fileSize int64, flag int, perm fs.FileMode, onClose func() error) (webdav.File, error)
	RemoveFile(fileName string) (bool, error)
	IsAllowedFileExtension(fileName string) bool
	RenameFile(fileName string, newFileName string) (bool, error)
}

type RemoteFileReader interface {
	OpenFile(name string, flag int, perm fs.FileMode, onClose func() error) (webdav.File, error)
	Stat(fileName string) (fs.FileInfo, error)
	IsAllowedFileExtension(fileName string) bool
	RenameFile(fileName string, newFileName string) (bool, error)
}
