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
	file              *os.File
	poster            string
	group             string
	cp                connectionpool.UsenetConnectionPool
	buffer            *segmentBuffer
	currentSize       int64
	modTime           time.Time
	wg                sync.WaitGroup
	onClose           func() error
	log               *slog.Logger
	closed            bool
	nzb               *nzb.Nzb
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
	tmpFileName := usenet.ReplaceFileExtension(fileName, ".nzb")
	f, err := os.OpenFile(tmpFileName, flag, perm)
	if err != nil {
		return nil, err
	}

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

	wf := &file{
		segments:     make([]nzb.NzbSegment, parts),
		parts:        parts,
		segmentSize:  segmentSize,
		fileSize:     fileSize,
		fileName:     fileName,
		fileNameHash: fileNameHash,
		file:         f,
		cp:           cp,
		poster:       poster,
		group:        randomGroup,
		buffer:       NewSegmentBuffer(segmentSize),
		log:          log,
		onClose:      onClose,
	}

	subject := fmt.Sprintf("[1/1] - \"%s\" yEnc (1/%d)", fileNameHash, parts)
	wf.nzb = &nzb.Nzb{
		Files: []nzb.NzbFile{
			{
				Segments: nzb.NzbSegmentSlice{},
				Subject:  subject,
				Groups:   []string{wf.group},
				Poster:   poster,
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(wf.currentSize, 10),
			"mod_time":       wf.modTime.Format(time.DateTime),
			"file_extension": filepath.Ext(wf.fileName),
			"file_name":      wf.fileName,
			"chunk_size":     strconv.FormatInt(wf.segmentSize, 10),
		},
	}

	// Create a timer that fires every 2 seconds
	updateTimer := time.NewTimer(2 * time.Second)

	// Start a goroutine that updates the metadata every time the timer fires
	go func() {

		for range updateTimer.C {
			if wf.closed {
				updateTimer.Stop()
				return
			}
			wf.updateNzbMetadata()
		}
	}()

	return wf, nil
}

func (u *file) Write(b []byte) (int, error) {
	n, err := u.buffer.Write(b)
	if err != nil {
		return n, err
	}
	if u.buffer.Size() == int(u.segmentSize) {
		u.addSegment(u.buffer.Bytes())
		u.buffer = NewSegmentBuffer(u.segmentSize)

		if n < len(b) {
			nb, err := u.buffer.Write(b[n:])
			if err != nil {
				return nb, err
			}

			n += nb
		}
	}

	u.currentSize += int64(n)
	u.modTime = time.Now()

	return n, nil
}

func (u *file) Close() error {
	u.closed = true
	// Upload the rest of segments
	if u.buffer.Size() > 0 {
		u.addSegment(u.buffer.Bytes())
	}

	// Wait for all uploads to finish
	u.wg.Wait()

	u.nzb.Files[0].Segments = u.segments
	u.nzb.Meta["file_size"] = strconv.FormatInt(u.currentSize, 10)
	u.nzb.Meta["mod_time"] = u.modTime.Format(time.DateTime)

	// Write and close the tmp nzb file
	u.file.Seek(0, 0)
	err := u.nzb.WriteIntoFile(u.file)
	if err != nil {
		return err
	}

	err = u.file.Close()
	if err != nil {
		return err
	}

	return u.onClose()
}

func (f *file) Fd() uintptr {
	return f.file.Fd()
}

func (f *file) Name() string {
	return f.file.Name()
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
	return NewFileInfo(metadata, f.file.Name())
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

func (f *file) updateNzbMetadata() error {
	m := f.getMetadata()

	f.nzb.Meta["file_size"] = strconv.FormatInt(m.FileSize, 10)
	f.nzb.Meta["mod_time"] = m.ModTime.Format(time.DateTime)
	f.file.Seek(0, 0)

	return f.nzb.WriteIntoFile(f.file)
}

func (u *file) getMetadata() usenet.Metadata {
	return usenet.Metadata{
		FileName:      u.fileName,
		ModTime:       u.modTime,
		FileSize:      u.currentSize,
		FileExtension: filepath.Ext(u.fileName),
		ChunkSize:     u.segmentSize,
	}
}

func (u *file) addSegment(b []byte) error {
	a := u.buildArticleData()
	na := NewNttpArticle(b, a)

	conn, err := u.cp.Get()
	if err != nil {
		u.log.Error("Error getting connection from pool.", "error", err)
		return err
	}
	u.wg.Add(1)
	go func(c *nntp.Conn, art *nntp.Article) {
		defer u.wg.Done()

		err := u.upload(art, c)
		if err != nil {
			u.log.Error("Error uploading segment.", "error", err, "segment", art.Header)
			return
		}

	}(conn, na)

	u.segments[u.currentPartNumber] = nzb.NzbSegment{
		Bytes:  a.partSize,
		Number: a.partNum,
		Id:     a.msgId,
	}

	u.currentPartNumber++
	return nil
}

func (u *file) buildArticleData() *ArticleData {
	start := u.currentPartNumber * u.segmentSize
	end := min((u.currentPartNumber+1)*u.segmentSize, u.fileSize)
	msgId := generateMessageId()

	return &ArticleData{
		partNum:   u.currentPartNumber + 1,
		partTotal: u.parts,
		partSize:  end - start,
		partBegin: start,
		partEnd:   end,
		fileNum:   1,
		fileTotal: 1,
		fileSize:  u.fileSize,
		fileName:  u.fileNameHash,
		poster:    u.poster,
		group:     u.group,
		msgId:     msgId,
	}
}

func (u *file) upload(a *nntp.Article, conn *nntp.Conn) error {
	defer u.cp.Free(conn)

	var err error
	for i := 0; i < 5; i++ {
		err = conn.Post(a)
		if err == nil {
			return nil
		} else {
			u.log.Error("Error uploading segment. Retrying", "error", err, "segment", a.Header)
		}
	}

	return err
}
