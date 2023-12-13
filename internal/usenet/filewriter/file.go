package filewriter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	status "github.com/javi11/usenet-drive/internal/usenet/statusreporter"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

var ErrUnexpectedFileSize = errors.New("file size does not match the expected size")

type nzbMetadata struct {
	fileNameHash     string
	filePath         string
	parts            int64
	group            string
	poster           string
	expectedFileSize int64
}

type file struct {
	io.ReaderFrom
	dryRun           bool
	nzbMetadata      nzbMetadata
	metadata         *usenet.Metadata
	cp               connectionpool.UsenetConnectionPool
	maxUploadRetries int
	onClose          func(err error) error
	log              *slog.Logger
	flag             int
	perm             fs.FileMode
	fs               osfs.FileSystem
	ctx              context.Context
	uploadErr        error
	sr               status.StatusReporter
	sessionId        uuid.UUID
}

func openFile(
	ctx context.Context,
	filePath string,
	flag int,
	perm fs.FileMode,
	fileSize int64,
	segmentSize int64,
	cp connectionpool.UsenetConnectionPool,
	randomGroup string,
	log *slog.Logger,
	maxUploadRetries int,
	dryRun bool,
	onClose func(err error) error,
	fs osfs.FileSystem,
	sr status.StatusReporter,
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

	fileNameHash := uuid.New().String()

	poster := generateRandomPoster()

	sessionId := uuid.New()
	sr.StartUpload(sessionId, filePath)

	return &file{
		ctx:              ctx,
		maxUploadRetries: maxUploadRetries,
		dryRun:           dryRun,
		cp:               cp,
		fs:               fs,
		log:              log.With("filename", fileName),
		onClose:          onClose,
		flag:             flag,
		perm:             perm,
		nzbMetadata: nzbMetadata{
			fileNameHash:     fileNameHash,
			filePath:         filePath,
			parts:            parts,
			group:            randomGroup,
			poster:           poster,
			expectedFileSize: fileSize,
		},
		metadata: &usenet.Metadata{
			FileName:      fileName,
			ModTime:       time.Now(),
			FileSize:      0,
			FileExtension: filepath.Ext(fileName),
			ChunkSize:     segmentSize,
		},
		sessionId: sessionId,
		sr:        sr,
	}, nil
}

func (f *file) ReadFrom(src io.Reader) (int64, error) {
	var bytesWritten int64
	wg := &multierror.Group{}
	segments := make([]*nzb.NzbSegment, f.nzbMetadata.parts)

	ctx, cancel := context.WithCancelCause(f.ctx)
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			err := wg.Wait().ErrorOrNil()
			if err != nil && !errors.Is(err, context.Canceled) {
				f.log.Error("Error closing upload threads.", "error", err)
			}

			f.sr.FinishUpload(f.sessionId)

			if err := context.Cause(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					f.log.Error("Error uploading the file", "error", err)
				}
				f.uploadErr = err

				return bytesWritten, err
			}

			return bytesWritten, nil
		default:
			buf := make([]byte, f.metadata.ChunkSize)
			bytesRead, err := f.readUntilBufferIsFull(src, buf)

			if bytesRead > 0 {
				if part := i + 1; part > int(f.nzbMetadata.parts) {
					f.log.Error(
						"Unexpected file size", "expected",
						f.nzbMetadata.expectedFileSize,
						"actual",
						bytesWritten,
						"expectedParts",
						f.nzbMetadata.parts,
						"actualParts",
						part,
					)
					cancel(ErrUnexpectedFileSize)

					continue
				}

				i := i
				retryErr := retry.Do(func() error {
					conn, err := f.cp.GetUploadConnection(ctx)
					if err != nil {
						if conn != nil {
							f.cp.Close(conn)
							conn = nil
						}

						return fmt.Errorf("error getting nntp connection: %w", err)
					}

					wg.Go(func() error {
						return f.addSegment(ctx, conn, segments, buf[0:bytesRead], i)
					})

					return nil
				},
					retry.Context(ctx),
					retry.Attempts(uint(f.maxUploadRetries)),
					retry.Delay(1*time.Second),
					retry.DelayType(retry.FixedDelay),
					retry.OnRetry(func(n uint, err error) {
						f.log.DebugContext(ctx, "Error getting connection for upload. Retrying", "error", err, "segment", i, "retry", n)
					}),
					retry.RetryIf(func(err error) bool {
						return nntpcli.IsRetryableError(err)
					}),
				)
				if retryErr != nil {
					err := retryErr
					var e retry.Error
					if errors.As(err, &e) {
						err = errors.Join(e.WrappedErrors()...)
					}

					cancel(err)
					continue
				}
				bytesWritten += int64(bytesRead)
				f.metadata.FileSize = bytesWritten
				f.metadata.ModTime = time.Now()

				f.sr.AddTimeData(f.sessionId, &status.TimeData{
					Milliseconds: time.Now().UnixNano() / 1e6,
					Bytes:        int64(bytesRead),
				})
			}
			if err != nil {
				// Upload was finished
				if err != io.EOF {
					// Upload finished but the file was not fully written
					f.log.Error("Error reading the file", "error", err)
					cancel(err)

					continue
				}

				if bytesWritten < f.nzbMetadata.expectedFileSize {
					f.log.Error(
						"Write end to early", "expected",
						f.nzbMetadata.expectedFileSize,
						"actual",
						bytesWritten,
						"expectedParts",
						f.nzbMetadata.parts,
						"actualParts",
						i+1,
					)
					cancel(io.ErrShortWrite)

					continue
				}

				if err := wg.Wait().ErrorOrNil(); err != nil {
					f.log.Error("Error uploading the file. The file will not be written.", "error", err)
					f.sr.FinishUpload(f.sessionId)
					f.uploadErr = err

					return bytesWritten, err
				}

				err := f.writeFinalNzb(segments)
				if err != nil {
					f.log.Error("Error writing the nzb file. The file will not be written.", "error", err)
					f.sr.FinishUpload(f.sessionId)
					f.uploadErr = err

					return bytesWritten, err
				}

				f.log.Info("Upload finished successfully.")
				f.sr.FinishUpload(f.sessionId)

				return bytesWritten, nil
			}
		}
	}
}

func (f *file) Write(b []byte) (int, error) {
	f.log.Error("Write not permitted. Use ReadFrom instead.")
	return 0, os.ErrPermission
}

func (f *file) Close() error {
	f.sr.FinishUpload(f.sessionId)

	return f.onClose(f.uploadErr)
}

func (f *file) Chdir() error {
	return os.ErrPermission
}

func (f *file) Chmod(mode os.FileMode) error {
	return os.ErrPermission
}

func (f *file) Chown(uid, gid int) error {
	return os.ErrPermission
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
	return *f.metadata
}

func (f *file) addSegment(ctx context.Context, conn connectionpool.Resource, segments []*nzb.NzbSegment, b []byte, segmentIndex int) error {
	log := f.log.With("segment_number", segmentIndex+1)

	err := retry.Do(func() error {
		a := f.buildArticleData(int64(segmentIndex))
		if a == nil {
			f.cp.Free(conn)
			conn = nil

			return fmt.Errorf("error building article data %w", ErrRetryable)
		}

		articleBytes, err := ArticleToBytes(b, a)
		if err != nil {
			log.Error("Error building article.", "error", err)
			f.cp.Free(conn)
			conn = nil

			return fmt.Errorf("error building article %w", ErrRetryable)
		}

		segments[segmentIndex] = &nzb.NzbSegment{
			Bytes:  a.partSize,
			Number: a.partNum,
			Id:     a.msgId,
		}

		// connection can be null in case OnRetry fails to get the connection
		if conn == nil {
			c, err := f.cp.GetUploadConnection(ctx)
			if err != nil {
				if conn != nil {
					f.cp.Close(conn)
					conn = nil
				}

				if errors.Is(err, context.Canceled) {
					return err
				}

				f.log.ErrorContext(ctx, "Error getting nntp connection:", "error", err, "segment", segmentIndex)

				return fmt.Errorf("error getting nntp connection: %w", err)
			}
			conn = c
		}

		if f.dryRun {
			time.Sleep(100 * time.Millisecond)
			f.cp.Free(conn)
			conn = nil

			return nil
		}

		nntpConn := conn.Value()
		err = nntpConn.Post(articleBytes)
		if err != nil {
			return fmt.Errorf("error posting article: %w", err)
		}

		f.cp.Free(conn)
		conn = nil

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(uint(f.maxUploadRetries)),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			l := log.With("retry", n)
			l.DebugContext(ctx, "Retrying upload", "error", err, "retry", n)

			if conn != nil {
				f.cp.Close(conn)
				conn = nil
			}

			c, e := f.cp.GetUploadConnection(ctx)
			if e != nil {
				if conn != nil {
					f.cp.Close(conn)
					conn = nil
				}

				f.log.InfoContext(ctx, "Error getting nntp connection:", "error", err, "segment", segmentIndex)
			}

			conn = c
		}),
		retry.RetryIf(func(err error) bool {
			return nntpcli.IsRetryableError(err) || errors.Is(err, ErrRetryable)
		}),
	)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			f.cp.Free(conn)
		} else if !errors.Is(err, net.ErrClosed) {
			f.cp.Close(conn)
		}
		conn = nil

		log.Error("Error uploading segment.", "error", err)
		return fmt.Errorf("error uploading segment, all retries exhausted. %w", err)
	}

	return nil
}

func (f *file) buildArticleData(segmentIndex int64) *ArticleData {
	start := segmentIndex * f.metadata.ChunkSize
	end := min((segmentIndex+1)*f.metadata.ChunkSize, f.nzbMetadata.expectedFileSize)
	msgId, err := generateMessageId()
	if err != nil {
		f.log.Error("Error generating message id.", "error", err)
		return nil
	}

	return &ArticleData{
		partNum:   segmentIndex + 1,
		partTotal: f.nzbMetadata.parts,
		partSize:  end - start,
		partBegin: start,
		partEnd:   end,
		fileNum:   1,
		fileTotal: 1,
		fileSize:  f.nzbMetadata.expectedFileSize,
		fileName:  f.nzbMetadata.fileNameHash,
		poster:    f.nzbMetadata.poster,
		group:     f.nzbMetadata.group,
		msgId:     msgId,
	}
}

func (f *file) writeFinalNzb(segments []*nzb.NzbSegment) error {
	for _, segment := range segments {
		if segment.Bytes == 0 {
			f.log.Warn("Upload was canceled. The file will not be written.")

			return io.ErrUnexpectedEOF
		}
	}

	// Create and upload the nzb file
	subject := fmt.Sprintf("[1/1] - \"%s\" yEnc (1/%d)", f.nzbMetadata.fileNameHash, f.nzbMetadata.parts)
	nzb := &nzb.Nzb{
		Files: []*nzb.NzbFile{
			{
				Segments: segments,
				Subject:  subject,
				Groups:   []string{f.nzbMetadata.group},
				Poster:   f.nzbMetadata.group,
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(f.metadata.FileSize, 10),
			"mod_time":       f.metadata.ModTime.Format(time.DateTime),
			"file_extension": filepath.Ext(f.metadata.FileName),
			"file_name":      f.metadata.FileName,
			"chunk_size":     strconv.FormatInt(f.metadata.ChunkSize, 10),
		},
	}

	// Write and close the tmp nzb file
	nzbFilePath := usenet.ReplaceFileExtension(f.nzbMetadata.filePath, ".nzb")
	b, err := nzb.ToBytes()
	if err != nil {
		f.log.Error("Malformed xml during nzb file writing.", "error", err)

		return io.ErrUnexpectedEOF
	}

	err = f.fs.WriteFile(nzbFilePath, b, f.perm)
	if err != nil {
		f.log.Error(fmt.Sprintf("Error writing the nzb file to %s.", nzbFilePath), "error", err)

		return io.ErrUnexpectedEOF
	}

	return nil
}

func (f *file) readUntilBufferIsFull(src io.Reader, buf []byte) (n int, err error) {
	for {
		if n >= len(buf) {
			return
		}
		nr, er := src.Read(buf[n:])
		err = er
		n += nr
		if err != nil {
			return
		}
	}
}
