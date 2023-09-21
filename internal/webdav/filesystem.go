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
	rclonecli "github.com/javi11/usenet-drive/pkg/rclone-cli"
	"golang.org/x/net/webdav"
)

type nzbFilesystem struct {
	rootPath            string
	tmpPath             string
	cn                  usenet.UsenetConnectionPool
	lock                sync.RWMutex
	queue               uploadqueue.UploadQueue
	log                 *slog.Logger
	uploadFileAllowlist []string
	nzbLoader           *usenet.NzbLoader
	rcloneCli           rclonecli.RcloneRcClient
	forceRefreshRclone  bool
}

func NewNzbFilesystem(
	rootPath string,
	tmpPath string,
	cn usenet.UsenetConnectionPool,
	queue uploadqueue.UploadQueue,
	log *slog.Logger,
	uploadFileAllowlist []string,
	nzbLoader *usenet.NzbLoader,
	rcloneCli rclonecli.RcloneRcClient,
	forceRefreshRclone bool,
) webdav.FileSystem {
	return &nzbFilesystem{
		rootPath:            rootPath,
		tmpPath:             tmpPath,
		cn:                  cn,
		queue:               queue,
		log:                 log,
		uploadFileAllowlist: uploadFileAllowlist,
		nzbLoader:           nzbLoader,
		rcloneCli:           rcloneCli,
		forceRefreshRclone:  forceRefreshRclone,
	}
}

func (fs *nzbFilesystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	if name = fs.resolve(name); name == "" {
		return os.ErrNotExist
	}

	err := os.Mkdir(name, perm)
	if err != nil {
		return err
	}

	fs.refreshRcloneCache(ctx, name)

	return nil
}

func (fs *nzbFilesystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}

	if isNzbFile(name) {
		// If file is a nzb file return a custom file that will mask the nzb
		return OpenNzbFile(ctx, name, flag, perm, fs.cn, fs.log, fs.nzbLoader)
	}

	originalName := getOriginalNzb(name)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		return OpenNzbFile(ctx, *originalName, flag, perm, fs.cn, fs.log, fs.nzbLoader)
	}

	onClose := func() {}
	if flag == os.O_RDWR|os.O_CREATE|os.O_TRUNC && utils.HasAllowedExtension(name, fs.uploadFileAllowlist) {
		tmpName := strings.Replace(name, fs.rootPath, fs.tmpPath, 1)
		// Prepare directory on tmp folder
		err := os.MkdirAll(filepath.Dir(tmpName), 0755)
		if err != nil {
			fs.log.ErrorContext(ctx, "Failed to create tmp folder", "err", err)
			return nil, err
		}
		// If the file is an allowed upload file, and was opened for writing, when close, add it to the upload queue
		onClose = func() {
			// Create a symlink to the original file on the tmp folder
			fs.log.DebugContext(ctx, "Creating symlink", "tmpName", tmpName, "name", name)
			err := os.Symlink(tmpName, name)
			if err != nil {
				fs.log.ErrorContext(ctx, "Failed to create symlink", "err", err)
			}

			fs.queue.AddJob(ctx, name)
			fs.refreshRcloneCache(ctx, name)
		}
		return OpenFile(tmpName, flag, perm, onClose, fs.log, fs.nzbLoader)
	}

	return OpenFile(name, flag, perm, onClose, fs.log, fs.nzbLoader)
}

func (fs *nzbFilesystem) RemoveAll(ctx context.Context, name string) error {
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

	if name == filepath.Clean(fs.rootPath) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}

	// Check if the file is a symlink
	if link, err := os.Readlink(name); err == nil {
		// If the file is a symlink, remove the original file
		if err := os.RemoveAll(link); err != nil {
			return err
		}
	}

	return os.RemoveAll(name)
}

func (fs *nzbFilesystem) Rename(ctx context.Context, oldName, newName string) error {
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
		// In case you want to update the file extension we need to update it in the original nzb file
		if filepath.Ext(newName) != filepath.Ext(oldName) {
			c, err := fs.nzbLoader.LoadFromFile(*originalName)
			if err != nil {
				return err
			}

			n := usenet.UpdateNzbMetadada(c.Nzb, usenet.UpdateableMetadata{
				FileExtension: filepath.Ext(newName),
			})
			b, err := usenet.NzbToBytes(n)
			if err != nil {
				return err
			}

			err = os.WriteFile(*originalName, b, 0766)
			if err != nil {
				return err
			}

			// Refresh the cache
			_, err = fs.nzbLoader.RefreshCachedNzb(*originalName, n)
			if err != nil {
				return err
			}
		}
		// If the file is a masked call the original nzb file
		oldName = *originalName
		newName = utils.ReplaceFileExtension(newName, ".nzb")

		if newName == oldName {
			return nil
		}
	}

	if root := filepath.Clean(fs.rootPath); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}

	err := os.Rename(oldName, newName)
	if err != nil {
		return err
	}

	fs.refreshRcloneCache(ctx, newName)

	return nil
}

func (fs *nzbFilesystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		// Filter metadata files
		return nil, os.ErrNotExist
	}

	if isNzbFile(name) {
		// If file is a nzb file return a custom file that will mask the nzb
		return NewNZBFileInfo(name, name, fs.log, fs.nzbLoader)
	}

	originalName := getOriginalNzb(name)
	if originalName != nil {
		// If the file is a masked call the original nzb file
		return NewNZBFileInfo(*originalName, name, fs.log, fs.nzbLoader)
	}

	// Build a new os.FileInfo with a mix of nzbFileInfo and metadata
	return os.Stat(name)
}

func (fs *nzbFilesystem) resolve(name string) string {
	// This implementation is based on Dir.Open's code in the standard net/http package.
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) ||
		strings.Contains(name, "\x00") {
		return ""
	}
	dir := fs.rootPath
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, filepath.FromSlash(slashClean(name)))
}

func (fs *nzbFilesystem) refreshRcloneCache(ctx context.Context, name string) {
	if fs.forceRefreshRclone {
		mountDir := filepath.Dir(strings.Replace(name, fs.rootPath, "", 1))
		if mountDir == "/" {
			mountDir = ""
		}
		err := fs.rcloneCli.RefreshCache(ctx, mountDir, true, false)
		if err != nil {
			fs.log.ErrorContext(ctx, "Failed to refresh cache", "err", err)
		}
	}
}
