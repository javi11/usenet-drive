package webdav

import (
	"context"
	"io/fs"

	"golang.org/x/net/webdav"
)

type RemoteFileWriter interface {
	OpenFile(ctx context.Context, name string, fileSize int64, flag int, perm fs.FileMode, onClose func(err error) error) (webdav.File, error)
	RemoveFile(ctx context.Context, fileName string) (bool, error)
	HasAllowedFileExtension(fileName string) bool
	RenameFile(ctx context.Context, fileName string, newFileName string) (bool, error)
}

type RemoteFileReader interface {
	OpenFile(ctx context.Context, name string, onClose func() error) (bool, webdav.File, error)
	Stat(fileName string) (bool, fs.FileInfo, error)
}
