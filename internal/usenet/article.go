package usenet

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/google/uuid"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

type Article struct {
	nttpArticle *nntp.Article
	NzbData     nzb.NzbFile
	Segment     nzb.NzbSegment
	FileName    string
}

type ArticleData struct {
	PartNum   int64
	PartTotal int64
	PartSize  int64
	PartBegin int64
	PartEnd   int64
	FileNum   int
	FileTotal int
	FileSize  int64
	FileName  string
}

type articleBuilder struct {
	poster string
	group  string
}

func NewArticleBuilder(poster, group string) *articleBuilder {
	return &articleBuilder{
		poster: poster,
		group:  group,
	}
}

func (ar *articleBuilder) NewArticle(p []byte, data *ArticleData) *Article {
	buf := new(bytes.Buffer)
	a := &nntp.Article{
		Header: map[string][]string{},
	}

	a.Header["From"] = []string{ar.poster}
	a.Header["Newsgroups"] = []string{ar.group}

	var msgid string
	t := time.Now()
	msgid = ar.generateMessageId()
	a.Header["Message-ID"] = []string{"<" + msgid + ">"}
	a.Header["X-Newsposter"] = []string{"UsenetDrive"}

	subj := fmt.Sprintf("[%d/%d] - \"%s\" yEnc (%d/%d)", data.FileNum, data.FileTotal, data.FileName, data.PartNum, data.PartTotal)
	a.Header["Subject"] = []string{subj}

	// yEnc begin line
	buf.WriteString(fmt.Sprintf("=ybegin part=%d total=%d line=128 size=%d name=%s\r\n", data.PartNum, data.PartTotal, data.FileSize, data.FileName))
	// yEnc part line
	buf.WriteString(fmt.Sprintf("=ypart begin=%d end=%d\r\n", data.PartBegin+1, data.PartEnd))

	// Encoded data
	yenc.Encode(p, buf)
	// yEnc end line
	h := crc32.NewIEEE()
	h.Write(p)
	buf.WriteString(fmt.Sprintf("=yend size=%d part=%d pcrc32=%08X\r\n", data.PartSize, data.PartNum, h.Sum32()))
	a.Body = buf
	// Nzb
	n := nzb.NzbFile{
		Groups:  []string{ar.group},
		Poster:  ar.poster,
		Date:    t.Unix(),
		Subject: subj,
	}
	s := nzb.NzbSegment{
		Bytes:  data.PartSize,
		Number: data.PartNum,
		Id:     msgid,
	}
	return &Article{nttpArticle: a, NzbData: n, Segment: s, FileName: data.FileName}
}

func (ar *articleBuilder) generateMessageId() string {
	id := uuid.New()
	return fmt.Sprintf("%s@usenetdrive", id.String())
}
