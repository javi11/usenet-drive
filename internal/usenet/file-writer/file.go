package usenetfilewriter

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/javi11/usenet-drive/internal/usenet"
	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type file struct {
	segments          []nzb.NzbSegment
	currentPartNumber int64
	parts             int64
	segmentSize       int64
	fileSize          int64
	fileName          string
	fileNameHash      string
	poster            string
	group             string
	cp                connectionpool.UsenetConnectionPool
	buffer            *segmentBuffer
	currentSize       int64
	modTime           time.Time
	wg                sync.WaitGroup
	onClose           func() error
	log               *slog.Logger
	flag              int
	perm              fs.FileMode
}

func openFile(
	_ context.Context,
	fileSize int64,
	segmentSize int64,
	fileName string,
	cp connectionpool.UsenetConnectionPool,
	randomGroup string,
	flag int,
	perm fs.FileMode,
	log *slog.Logger,
	onClose func() error,
) (*file, error) {
	parts := fileSize / segmentSize
	rem := fileSize % segmentSize
	if rem > 0 {
		parts++
	}

	fileNameHash, err := generateHashFromString(fileName)
	if err != nil {
		return nil, err
	}

	poster := generateRandomPoster()

	return &file{
		segments:     make([]nzb.NzbSegment, parts),
		parts:        parts,
		segmentSize:  segmentSize,
		fileSize:     fileSize,
		fileName:     fileName,
		fileNameHash: fileNameHash,
		cp:           cp,
		poster:       poster,
		group:        randomGroup,
		buffer:       NewSegmentBuffer(segmentSize),
		log:          log,
		onClose:      onClose,
		flag:         flag,
		perm:         perm,
	}, nil
}

func (f *file) Write(b []byte) (int, error) {
	n, err := f.buffer.Write(b)
	if err != nil {
		return n, err
	}
	if f.buffer.Size() == int(f.segmentSize) {
		f.addSegment(f.buffer.Bytes())
		f.buffer = NewSegmentBuffer(f.segmentSize)

		if n < len(b) {
			nb, err := f.buffer.Write(b[n:])
			if err != nil {
				return nb, err
			}

			n += nb
		}
	}

	f.currentSize += int64(n)
	f.modTime = time.Now()

	return n, nil
}

func (f *file) Close() error {
	// Upload the rest of segments
	if f.buffer.Size() > 0 {
		f.addSegment(f.buffer.Bytes())
	}

	// Wait for all uploads to finish
	f.wg.Wait()

	subject := fmt.Sprintf("[1/1] - \"%s\" yEnc (1/%d)", f.fileNameHash, f.parts)
	nzb := nzb.Nzb{
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
	name := usenet.ReplaceFileExtension(f.fileName, ".nzb")
	fl, err := os.OpenFile(name, f.flag, f.perm)
	if err != nil {
		return err
	}

	err = nzb.WriteIntoFile(fl)
	if err != nil {
		return err
	}

	err = fl.Close()
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

func (f *file) addSegment(b []byte) error {
	a := f.buildArticleData()
	na := NewNttpArticle(b, a)

	conn, err := f.cp.Get()
	if err != nil {
		f.log.Error("Error getting connection from pool.", "error", err)
		return err
	}
	f.wg.Add(1)
	go func(c *nntp.Conn, art *nntp.Article) {
		defer f.wg.Done()

		err := f.upload(art, c)
		if err != nil {
			f.log.Error("Error uploading segment.", "error", err, "segment", art.Header)
			return
		}

	}(conn, na)

	f.segments[f.currentPartNumber] = nzb.NzbSegment{
		Bytes:  a.partSize,
		Number: a.partNum,
		Id:     a.msgId,
	}

	f.currentPartNumber++
	return nil
}

func (f *file) buildArticleData() *ArticleData {
	start := f.currentPartNumber * f.segmentSize
	end := min((f.currentPartNumber+1)*f.segmentSize, f.fileSize)
	msgId := generateMessageId()

	return &ArticleData{
		partNum:   f.currentPartNumber + 1,
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

func (f *file) upload(a *nntp.Article, conn *nntp.Conn) error {
	defer f.cp.Free(conn)

	var err error
	for i := 0; i < 5; i++ {
		err = conn.Post(a)
		if err == nil {
			return nil
		} else {
			f.log.Error("Error uploading segment. Retrying", "error", err, "segment", a.Header)
		}
	}

	return err
}
