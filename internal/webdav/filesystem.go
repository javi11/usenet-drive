package webdav

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/utils"
	"golang.org/x/net/webdav"
)

type nzbFilesystem struct {
	root                string
	cn                  usenet.UsenetConnectionPool
	lock                *sync.RWMutex
	queue               uploadqueue.UploadQueue
	log                 *slog.Logger
	uploadFileWhitelist []string
}

func NewNzbFilesystem(
	root string,
	cn usenet.UsenetConnectionPool,
	queue uploadqueue.UploadQueue,
	log *slog.Logger,
	uploadFileWhitelist []string,
) webdav.FileSystem {
	return nzbFilesystem{
		root:                root,
		cn:                  cn,
		lock:                &sync.RWMutex{},
		queue:               queue,
		log:                 log,
		uploadFileWhitelist: uploadFileWhitelist,
	}
}

func (fs nzbFilesystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	if name = fs.resolve(name); name == "" {
		return os.ErrNotExist
	}
	return os.Mkdir(name, perm)
}

func (fs nzbFilesystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}

	if isNzbFile(name) {
		// If file is a nzb file return a custom file that will mask the nzb
		return OpenNzbFile(ctx, name, flag, perm, fs.cn, fs.lock, fs.log)
	}

	originalName := getOriginalNzb(name)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		return OpenNzbFile(ctx, *originalName, flag, perm, fs.cn, fs.lock, fs.log)
	}

	onClose := func() {}
	if flag == os.O_RDWR|os.O_CREATE|os.O_TRUNC && fs.hasAllowedExtension(name, fs.uploadFileWhitelist) {
		// If the file is an allowed upload file, and was opened for writing, when close, add it to the upload queue
		onClose = func() {
			fs.queue.AddJob(ctx, name)
		}
	}

	return OpenFile(name, flag, perm, fs.root, fs.lock, onClose, fs.log)
}

func (fs nzbFilesystem) RemoveAll(ctx context.Context, name string) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	if name = fs.resolve(name); name == "" {
		return os.ErrNotExist
	}

	originalName := getOriginalNzb(name)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		name = *originalName
	}

	if name == filepath.Clean(fs.root) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}
	return os.RemoveAll(name)
}

func (fs nzbFilesystem) Rename(ctx context.Context, oldName, newName string) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if oldName = fs.resolve(oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = fs.resolve(newName); newName == "" {
		return os.ErrNotExist
	}

	originalName := getOriginalNzb(oldName)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		oldName = *originalName
		newName = utils.ReplaceFileExtension(newName, ".nzb")
	}

	if root := filepath.Clean(fs.root); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}
	return os.Rename(oldName, newName)
}

func (fs nzbFilesystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		// Filter metadata files
		return nil, os.ErrNotExist
	}

	if isNzbFile(name) {
		// If file is a nzb file return a custom file that will mask the nzb
		return NewFileInfoWithMetadata(name, fs.log)
	}

	originalName := getOriginalNzb(name)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		return NewFileInfoWithMetadata(*originalName, fs.log)
	}

	// Build a new os.FileInfo with a mix of nzbFileInfo and metadata
	return os.Stat(name)
}

func (fs nzbFilesystem) resolve(name string) string {
	// This implementation is based on Dir.Open's code in the standard net/http package.
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) ||
		strings.Contains(name, "\x00") {
		return ""
	}
	dir := fs.root
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, filepath.FromSlash(slashClean(name)))
}

func (fs nzbFilesystem) hasAllowedExtension(path string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
