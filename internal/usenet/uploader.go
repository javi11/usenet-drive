package usenet

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type Uploader interface {
	NewFileUploader(name string, fileSize int64) (*fileUploader, error)
}

type uploader struct {
	segmentSize int64
	cp          UsenetConnectionPool
	postGroups  []string
}

func NewUploader(segmentSize int64, cp UsenetConnectionPool, postGroups []string) Uploader {
	return &uploader{
		segmentSize: segmentSize,
		cp:          cp,
		postGroups:  postGroups,
	}
}

func (u *uploader) NewFileUploader(fileName string, fileSize int64) (*fileUploader, error) {
	parts := fileSize / u.segmentSize
	rem := fileSize % u.segmentSize
	if rem > 0 {
		parts++
	}

	randomGroup := u.postGroups[rand.Intn(len(u.postGroups))]
	poster := u.generatePoster()
	fileNameHash, err := generateHashFromString(fileName)
	if err != nil {
		return nil, err
	}

	return &fileUploader{
		segments:       make([]nzb.NzbSegment, parts),
		parts:          parts,
		segmentSize:    u.segmentSize,
		fileSize:       fileSize,
		fileName:       fileName,
		fileNameHash:   fileNameHash,
		cp:             u.cp,
		poster:         poster,
		group:          randomGroup,
		articleBuilder: NewArticleBuilder(poster, randomGroup),
		buffer:         NewUploadBuffer(u.segmentSize),
	}, nil
}

func (u *uploader) generatePoster() string {
	email := faker.Email()
	username := faker.Username()

	return fmt.Sprintf("%s <%s>", username, email)
}

type FileUploader interface {
	Write(b []byte) (int, error)
	Build() *nzb.Nzb
	GetMetadata() Metadata
	Close()
}

type fileUploader struct {
	segments       []nzb.NzbSegment
	partnum        int64
	parts          int64
	segmentSize    int64
	fileSize       int64
	fileName       string
	fileNameHash   string
	poster         string
	group          string
	articleBuilder *articleBuilder
	cp             UsenetConnectionPool
	buffer         *uploadBuffer
	currentSize    int64
	modTime        time.Time
}

func (u *fileUploader) Write(b []byte) (int, error) {
	n, err := u.buffer.Write(b)
	if err != nil {
		return n, err
	}
	if u.buffer.Size() == int(u.segmentSize) {
		u.addSegment(u.buffer.Dump())

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

func (u *fileUploader) Build() *nzb.Nzb {
	if u.buffer.Size() > 0 {
		u.addSegment(u.buffer.Dump())
	}

	fileExtension := filepath.Ext(u.fileName)
	// [fileNumber/fileTotal] - "fileName" yEnc (partNumber/partTotal)
	subject := fmt.Sprintf("[1/1] - \"%s\" yEnc (1/%d)", u.fileNameHash, u.parts)
	nzb := &nzb.Nzb{
		Files: []nzb.NzbFile{
			{
				Subject:  subject,
				Segments: u.segments,
				Groups:   []string{u.group},
				Poster:   u.poster,
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(u.fileSize, 10),
			"mod_time":       time.Now().Format(time.DateTime),
			"file_extension": fileExtension,
			"file_name":      u.fileName,
			"chunk_size":     strconv.FormatInt(u.segmentSize, 10),
		},
	}

	return nzb
}

func (u *fileUploader) Close() {
	u.buffer.Clear()
}

func (u *fileUploader) newArticle(b []byte) (*Article, error) {
	start := u.partnum * u.segmentSize
	end := min((u.partnum+1)*u.segmentSize, u.fileSize)

	ad := &ArticleData{
		PartNum:   u.partnum + 1,
		PartTotal: u.parts,
		PartSize:  end - start,
		PartBegin: start,
		PartEnd:   end,
		FileNum:   1,
		FileTotal: 1,
		FileSize:  u.fileSize,
		FileName:  u.fileNameHash,
	}
	a := u.articleBuilder.NewArticle(b, ad)
	u.partnum++

	return a, nil
}

func (u *fileUploader) GetMetadata() Metadata {
	return Metadata{
		FileName:      u.fileName,
		ModTime:       u.modTime,
		FileSize:      u.currentSize,
		FileExtension: filepath.Ext(u.fileName),
		ChunkSize:     u.segmentSize,
	}
}

func (u *fileUploader) addSegment(b []byte) error {
	a, err := u.newArticle(b)
	if err != nil {
		return err
	}

	err = u.upload(a)
	if err != nil {
		return err
	}

	u.segments[u.partnum-1] = a.Segment

	return nil
}

func (u *fileUploader) upload(a *Article) error {
	conn, err := u.cp.Get()
	if err != nil {
		return err
	}
	defer u.cp.Free(conn)

	return conn.Post(a.nttpArticle)
}
