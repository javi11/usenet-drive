package usenet

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/go-faker/faker/v4"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type Uploader interface {
	NewFileUploader(name string, fileSize int64) *fileUploader
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

func (u *uploader) NewFileUploader(fileName string, fileSize int64) *fileUploader {
	parts := fileSize / u.segmentSize
	rem := fileSize % u.segmentSize
	if rem > 0 {
		parts++
	}

	randomGroup := u.postGroups[rand.Intn(len(u.postGroups))]
	poster := u.generatePoster()

	return &fileUploader{
		segments:       make([]nzb.NzbSegment, parts),
		parts:          parts,
		segmentSize:    u.segmentSize,
		fileSize:       fileSize,
		fileName:       fileName,
		cp:             u.cp,
		poster:         poster,
		group:          randomGroup,
		articleBuilder: NewArticleBuilder(poster, randomGroup),
	}
}

func (u *uploader) generatePoster() string {
	email := faker.Email()
	username := faker.Username()

	return fmt.Sprintf("%s <%s>", username, email)
}

type fileUploader struct {
	segments       []nzb.NzbSegment
	partnum        int64
	parts          int64
	segmentSize    int64
	fileSize       int64
	fileName       string
	poster         string
	group          string
	articleBuilder *articleBuilder
	cp             UsenetConnectionPool
}

func (u *fileUploader) AddSegment(b []byte) error {
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

func (u *fileUploader) Build() *nzb.Nzb {
	fileExtension := filepath.Ext(u.fileName)

	nzb := &nzb.Nzb{
		Files: []nzb.NzbFile{
			{
				Segments: u.segments,
				Groups:   []string{u.group},
				Poster:   u.poster,
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(u.fileSize, 10),
			"mod_time":       time.Now().Format(time.RFC3339),
			"file_extension": fileExtension,
			"file_name":      u.fileName,
			"chunk_size":     strconv.FormatInt(u.segmentSize, 10),
		},
	}

	return nzb
}

func (u *fileUploader) newArticle(b []byte) (*Article, error) {
	start := u.partnum * u.segmentSize
	end := min((u.partnum+1)*u.segmentSize, u.fileSize)

	fileNameHash, err := u.generateHashName(u.fileName)
	if err != nil {
		return nil, err
	}

	ad := &ArticleData{
		PartNum:   u.partnum + 1,
		PartTotal: u.parts,
		PartSize:  end - start,
		PartBegin: start,
		PartEnd:   end,
		FileNum:   1,
		FileTotal: 1,
		FileSize:  u.fileSize,
		FileName:  fileNameHash,
	}
	a := u.articleBuilder.NewArticle(b, ad)
	u.partnum++

	return a, nil
}

func (u *fileUploader) upload(a *Article) error {
	conn, err := u.cp.Get()
	if err != nil {
		return err
	}
	defer u.cp.Free(conn)

	return conn.Post(&nntp.Article{
		Body: a.Body,
	})
}

func (u *fileUploader) generateHashName(fileName string) (string, error) {
	hash := md5.Sum([]byte(fileName))
	return hex.EncodeToString(hash[:]), nil
}
