package filereader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
)

type file struct {
	name      string
	buffer    *buffer
	innerFile *os.File
	fsMutex   sync.RWMutex
	log       *slog.Logger
	metadata  usenet.Metadata
	nzbLoader *nzbloader.NzbLoader
	onClose   func() error
	cNzb      corruptednzbsmanager.CorruptedNzbsManager
}

func openFile(
	ctx context.Context,
	name string,
	flag int,
	perm os.FileMode,
	cp connectionpool.UsenetConnectionPool,
	log *slog.Logger,
	onClose func() error,
	nzbLoader *nzbloader.NzbLoader,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
) (bool, *file, error) {
	if !isNzbFile(name) {
		originalName := getOriginalNzb(name)
		if originalName != "" {
			// If the file is a masked call the original nzb file
			name = originalName
		} else {
			return false, nil, nil
		}
	}

	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return true, nil, err
	}

	n, err := nzbLoader.LoadFromFileReader(f)
	if err != nil {
		log.ErrorContext(ctx, fmt.Sprintf("Error getting loading nzb %s", name), "err", err)
		return true, nil, os.ErrNotExist
	}

	buffer, err := NewBuffer(&n.Nzb.Files[0], int(n.Metadata.FileSize), int(n.Metadata.ChunkSize), cp, log)
	if err != nil {
		return true, nil, err
	}

	return true, &file{
		innerFile: f,
		buffer:    buffer,
		metadata:  n.Metadata,
		name:      usenet.ReplaceFileExtension(name, n.Metadata.FileExtension),
		log:       log,
		nzbLoader: nzbLoader,
		onClose:   onClose,
		cNzb:      cNzb,
	}, nil
}

func (f *file) Chdir() error {
	return f.innerFile.Chdir()
}

func (f *file) Chmod(mode os.FileMode) error {
	return f.innerFile.Chmod(mode)
}

func (f *file) Chown(uid, gid int) error {
	return f.innerFile.Chown(uid, gid)
}

func (f *file) Close() error {
	if err := f.buffer.Close(); err != nil {
		return err
	}

	if f.onClose != nil {
		if err := f.onClose(); err != nil {
			return err
		}
	}

	return f.innerFile.Close()
}

func (f *file) Fd() uintptr {
	return f.innerFile.Fd()
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Read(b []byte) (int, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	n, err := f.buffer.Read(b)
	if err != nil {
		if errors.Is(err, ErrCorruptedNzb) {
			f.log.Error("Marking file as corrupted:", "error", err, "fileName", f.name)
			err := f.cNzb.Add(context.Background(), f.name, err.Error())
			if err != nil {
				f.log.Error("Error adding corrupted nzb to the database:", "error", err)
			}

			return n, io.ErrUnexpectedEOF
		}

		return n, err
	}

	return n, nil
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	n, err := f.buffer.ReadAt(b, off)
	if err != nil {
		if errors.Is(err, ErrCorruptedNzb) {
			f.log.Error("Marking file as corrupted:", "error", err, "fileName", f.name)
			err := f.cNzb.Add(context.Background(), f.name, err.Error())
			if err != nil {
				f.log.Error("Error adding corrupted nzb to the database:", "error", err)
			}

			return n, io.ErrUnexpectedEOF
		}

		return n, err
	}

	return n, nil
}

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	infos, err := f.innerFile.Readdir(n)
	if err != nil {
		return nil, err
	}

	var merr multierror.Group

	for i, info := range infos {
		name := info.Name()
		i := i
		if !isNzbFile(name) {
			originalName := getOriginalNzb(info.Name())
			if originalName != "" {
				// If the file is a masked call the original nzb file
				name = originalName
			} else {
				infos[i] = info
				continue
			}
		}

		merr.Go(func() error {
			n := filepath.Join(f.innerFile.Name(), name)
			inf, err := NewFileInfo(
				n,
				f.log,
				f.nzbLoader,
			)
			if err != nil {
				infos[i] = nil
				return err
			}

			infos[i] = inf

			return nil
		})

	}

	if err := merr.Wait(); err != nil {
		f.log.Error("error reading remote directory", "error", err)

		// Remove nulls from infos
		var filteredInfos []os.FileInfo
		for _, info := range infos {
			if info != nil {
				filteredInfos = append(filteredInfos, info)
			}
		}

		return filteredInfos, nil
	}

	return infos, nil
}

func (f *file) Readdirnames(n int) ([]string, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()
	return f.innerFile.Readdirnames(n)
}

func (f *file) Seek(offset int64, whence int) (n int64, err error) {
	f.fsMutex.RLock()
	n, err = f.buffer.Seek(offset, whence)
	f.fsMutex.RUnlock()
	return
}

func (f *file) SetDeadline(t time.Time) error {
	return f.innerFile.SetDeadline(t)
}

func (f *file) SetReadDeadline(t time.Time) error {
	return f.innerFile.SetReadDeadline(t)
}

func (f *file) SetWriteDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *file) Stat() (os.FileInfo, error) {
	f.fsMutex.RLock()
	defer f.fsMutex.RUnlock()

	return NeFileInfoWithMetadata(f.metadata, f.innerFile.Name())
}

func (f *file) Sync() error {
	return f.innerFile.Sync()
}

func (f *file) Truncate(size int64) error {
	return os.ErrPermission
}

func (f *file) Write(b []byte) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}
