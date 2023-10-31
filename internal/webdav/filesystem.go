package webdav

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/javi11/usenet-drive/pkg/rclonecli"
	"golang.org/x/net/webdav"
)

type remoteFilesystem struct {
	rootPath           string
	lock               sync.RWMutex
	log                *slog.Logger
	rcloneCli          rclonecli.RcloneRcClient
	forceRefreshRclone bool
	fileWriter         RemoteFileWriter
	fileReader         RemoteFileReader
}

func NewRemoteFilesystem(
	rootPath string,
	fileWriter RemoteFileWriter,
	fileReader RemoteFileReader,
	rcloneCli rclonecli.RcloneRcClient,
	forceRefreshRclone bool,
	log *slog.Logger,
) webdav.FileSystem {
	return &remoteFilesystem{
		rootPath:           rootPath,
		log:                log,
		fileWriter:         fileWriter,
		fileReader:         fileReader,
		forceRefreshRclone: forceRefreshRclone,
		rcloneCli:          rcloneCli,
	}
}

func (fs *remoteFilesystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
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

func (fs *remoteFilesystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}

	ok, f, err := fs.fileReader.OpenFile(ctx, name, flag, perm, nil)
	if err != nil {
		return nil, err
	}

	if ok {
		// Return the file in case it was found in the remote
		return f, nil
	}

	onClose := func(err error) error {
		return nil
	}
	if flag == os.O_RDWR|os.O_CREATE|os.O_TRUNC && fs.fileWriter.HasAllowedFileExtension(name) {
		finalSize, err := strconv.ParseInt(ctx.Value(reqContentLengthKey).(string), 10, 64)
		if err != nil {
			return nil, err
		}

		// If the file is an allowed upload file, and was opened for writing, when close, add it to the upload queue
		onClose = func(err error) error {
			if err != nil {
				fs.log.InfoContext(ctx, "Upload file was discarded because an error", "name", name, "err", err)
				return nil
			}

			fs.log.InfoContext(ctx, "File uploaded", "name", name, "size", finalSize)
			fs.refreshRcloneCache(ctx, name)
			return nil
		}

		fs.log.InfoContext(ctx, "Uploading file", "name", name, "size", finalSize)
		return fs.fileWriter.OpenFile(ctx, name, finalSize, flag, perm, onClose)
	}

	return OpenFile(name, flag, perm, onClose, fs.log, fs.fileReader)
}

func (fs *remoteFilesystem) RemoveAll(ctx context.Context, name string) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	if name = fs.resolve(name); name == "" {
		return os.ErrNotExist
	}

	ok, err := fs.fileWriter.RemoveFile(ctx, name)
	if err != nil {
		return err
	}

	if ok {
		fs.refreshRcloneCache(ctx, name)
		return nil
	}

	if name == filepath.Clean(fs.rootPath) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}

	err = os.RemoveAll(name)
	if err != nil {
		return err
	}

	fs.refreshRcloneCache(ctx, name)

	return nil
}

func (fs *remoteFilesystem) Rename(ctx context.Context, oldName, newName string) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	if oldName = fs.resolve(oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = fs.resolve(newName); newName == "" {
		return os.ErrNotExist
	}

	ok, err := fs.fileWriter.RenameFile(ctx, oldName, newName)
	if err != nil {
		return err
	}

	if ok {
		fs.refreshRcloneCache(ctx, newName)
		return nil
	}

	if root := filepath.Clean(fs.rootPath); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}

	err = os.Rename(oldName, newName)
	if err != nil {
		return err
	}

	fs.refreshRcloneCache(ctx, newName)

	return nil
}

func (fs *remoteFilesystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	if name = fs.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}

	stat, e := os.Stat(name)
	if e != nil {
		if !os.IsNotExist(e) {
			return nil, e
		}
	}

	if stat != nil && stat.IsDir() {
		return stat, nil
	}

	ok, s, err := fs.fileReader.Stat(name)
	if err != nil {
		return nil, err
	}

	if ok {
		// Return the remote file info if it exists
		return s, nil
	}

	return stat, e
}

func (fs *remoteFilesystem) resolve(name string) string {
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

func (fs *remoteFilesystem) refreshRcloneCache(ctx context.Context, name string) {
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
