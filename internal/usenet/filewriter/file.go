package filewriter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

var ErrUnexpectedFileSize = errors.New("file size does not match the expected size")

type file struct {
	io.ReaderFrom
	dryRun           bool
	segments         []nzb.NzbSegment
	parts            int64
	segmentSize      int64
	fileSize         int64
	fileName         string
	fileNameHash     string
	filePath         string
	poster           string
	group            string
	cp               connectionpool.UsenetConnectionPool
	maxUploadRetries int
	currentSize      int64
	modTime          time.Time
	merr             *multierror.Group
	onClose          func() error
	log              *slog.Logger
	flag             int
	perm             fs.FileMode
	nzbLoader        nzbloader.NzbLoader
}

func openFile(
	ctx context.Context,
	fileSize int64,
	segmentSize int64,
	filePath string,
	cp connectionpool.UsenetConnectionPool,
	randomGroup string,
	flag int,
	perm fs.FileMode,
	log *slog.Logger,
	nzbLoader nzbloader.NzbLoader,
	dryRun bool,
	onClose func() error,
) (*file, error) {
	if dryRun {
		log.InfoContext(ctx, "Dry run. Skipping upload", "filename", filePath)
	}

	parts := fileSize / segmentSize
	rem := fileSize % segmentSize
	if rem > 0 {
		parts++
	}

	fileName := filepath.Base(filePath)

	fileNameHash, err := generateHashFromString(fileName)
	if err != nil {
		return nil, err
	}

	poster := generateRandomPoster()

	return &file{
		maxUploadRetries: 5,
		dryRun:           dryRun,
		segments:         make([]nzb.NzbSegment, parts),
		parts:            parts,
		segmentSize:      segmentSize,
		fileSize:         fileSize,
		fileName:         fileName,
		filePath:         filePath,
		fileNameHash:     fileNameHash,
		cp:               cp,
		poster:           poster,
		group:            randomGroup,
		log:              log.With("filename", fileName),
		onClose:          onClose,
		flag:             flag,
		perm:             perm,
		nzbLoader:        nzbLoader,
		merr:             &multierror.Group{},
	}, nil
}

func (f *file) ReadFrom(src io.Reader) (written int64, err error) {
	for i := 0; ; i++ {
		if i+1 > int(f.parts) {
			f.log.Error("Unexpected file size", "expected", f.fileSize, "actual", written, "expectedParts", f.parts, "actualParts", i)
			err = ErrUnexpectedFileSize
			break
		}

		buf := make([]byte, f.segmentSize)
		nr, er := src.Read(buf)
		if nr > 0 {
			err = f.addSegment(buf[0:nr], i, f.maxUploadRetries)
			if err != nil {
				return written, err
			}
			written += int64(nr)
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return written, err
}

func (f *file) Write(b []byte) (int, error) {
	f.log.Error("Write not permitted. Use ReadFrom instead.")
	return 0, os.ErrPermission
}

func (f *file) Close() error {
	// Wait for all uploads to finish
	if err := f.merr.Wait().ErrorOrNil(); err != nil {
		f.log.Error("Error uploading the file. The file will not be written.", "fileName", f.fileName, "error", err)

		return io.ErrUnexpectedEOF
	}

	for _, segment := range f.segments {
		if segment.Bytes == 0 {
			f.log.Warn("Upload was canceled. The file will not be written.", "fileName", f.fileName)

			return io.ErrUnexpectedEOF
		}
	}

	// Create and upload the nzb file
	subject := fmt.Sprintf("[1/1] - \"%s\" yEnc (1/%d)", f.fileNameHash, f.parts)
	nzb := &nzb.Nzb{
		Files: []nzb.NzbFile{
			{
				Segments: f.segments,
				Subject:  subject,
				Groups:   []string{f.group},
				Poster:   f.poster,
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(f.currentSize, 10),
			"mod_time":       f.modTime.Format(time.DateTime),
			"file_extension": filepath.Ext(f.fileName),
			"file_name":      f.fileName,
			"chunk_size":     strconv.FormatInt(f.segmentSize, 10),
		},
	}

	// Write and close the tmp nzb file
	nzbFilePath := usenet.ReplaceFileExtension(f.filePath, ".nzb")
	b, err := nzb.ToBytes()
	if err != nil {
		return err
	}

	err = os.WriteFile(nzbFilePath, b, f.perm)
	if err != nil {
		return err
	}

	_, err = f.nzbLoader.RefreshCachedNzb(nzbFilePath, nzb)
	if err != nil {
		return err
	}

	return f.onClose()
}

func (f *file) Fd() uintptr {
	return 0
}

func (f *file) Name() string {
	return f.getMetadata().FileName
}

func (f *file) Read(b []byte) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) Readdir(n int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, os.ErrPermission
}

func (f *file) Readdirnames(n int) ([]string, error) {
	return []string{}, os.ErrPermission
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return 0, os.ErrPermission
}

func (f *file) SetDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *file) SetReadDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *file) SetWriteDeadline(t time.Time) error {
	return os.ErrPermission
}

func (f *file) Stat() (os.FileInfo, error) {
	metadata := f.getMetadata()
	return NewFileInfo(metadata, metadata.FileName)
}

func (f *file) Sync() error {
	return os.ErrPermission
}

func (f *file) Truncate(size int64) error {
	return os.ErrPermission
}

func (f *file) WriteAt(b []byte, off int64) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) WriteString(s string) (int, error) {
	return 0, os.ErrPermission
}

func (f *file) getMetadata() usenet.Metadata {
	return usenet.Metadata{
		FileName:      f.fileName,
		ModTime:       f.modTime,
		FileSize:      f.currentSize,
		FileExtension: filepath.Ext(f.fileName),
		ChunkSize:     f.segmentSize,
	}
}

func (f *file) addSegment(b []byte, segmentIndex int, retries int) error {
	conn, err := f.cp.Get()
	if err != nil {
		if conn != nil {
			if err = f.cp.Close(conn); err != nil {
				f.log.Error("Error closing connection.", "error", err)
			}
		}
		f.log.Error("Error getting connection from pool.", "error", err)

		if retries > 0 {
			return f.addSegment(b, segmentIndex, retries-1)
		}

		return err
	}

	f.merr.Go(func() error {
		defer func() {
			err := f.cp.Free(conn)
			if err != nil {
				f.log.Debug("Error freeing the connection.", "error", err)
			}
		}()

		a := f.buildArticleData(int64(segmentIndex))
		na, err := NewNttpArticle(b, a)
		if err != nil {
			f.log.Error("Error building article.", "error", err, "segment", a)
			return err
		}

		f.segments[segmentIndex] = nzb.NzbSegment{
			Bytes:  a.partSize,
			Number: a.partNum,
			Id:     a.msgId,
		}

		err = f.upload(na, conn)
		if err != nil {
			f.log.Error("Error uploading segment.", "error", err, "segment", na.Header)
			return err
		}

		return nil

	})

	return nil
}

func (f *file) buildArticleData(segmentIndex int64) *ArticleData {
	start := segmentIndex * f.segmentSize
	end := min((segmentIndex+1)*f.segmentSize, f.fileSize)
	msgId := generateMessageId()

	return &ArticleData{
		partNum:   segmentIndex + 1,
		partTotal: f.parts,
		partSize:  end - start,
		partBegin: start,
		partEnd:   end,
		fileNum:   1,
		fileTotal: 1,
		fileSize:  f.fileSize,
		fileName:  f.fileNameHash,
		poster:    f.poster,
		group:     f.group,
		msgId:     msgId,
	}
}

func (f *file) upload(a *nntp.Article, conn connectionpool.NntpConnection) error {
	if f.dryRun {
		time.Sleep(100 * time.Millisecond)

		return nil
	}

	var err error
	for i := 0; i < f.maxUploadRetries; i++ {
		err = conn.Post(a)
		if err == nil {
			return nil
		}

		if !connectionpool.IsRetryable(err) {
			f.log.Error("Error uploading segment.", "error", err, "segment", a.Header, "retryable", false)
			return err
		}

		f.log.Error("Error uploading segment. Retrying", "error", err, "segment", a.Header, "retryable", true)
		err := f.cp.Close(conn)
		if err != nil {
			f.log.Error("Error closing connection.", "error", err)
		}
		conn, err = f.cp.Get()
		if err != nil {
			f.log.Error("Error getting connection from pool.", "error", err)
			return err
		}
		defer func() {
			err := f.cp.Free(conn)
			if err != nil {
				f.log.Error("Error freeing the connection.", "error", err)
			}
		}()

	}

	return err
}
