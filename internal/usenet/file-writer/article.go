package usenetfilewriter

import (
	"bytes"
	"fmt"
	"hash/crc32"

	"github.com/chrisfarms/nntp"
	"github.com/javi11/usenet-drive/pkg/yenc"
)

type ArticleData struct {
	partNum   int64
	partTotal int64
	partSize  int64
	partBegin int64
	partEnd   int64
	fileNum   int
	fileTotal int
	fileSize  int64
	fileName  string
	poster    string
	group     string
	msgId     string
}

func NewNttpArticle(p []byte, data *ArticleData) *nntp.Article {
	buf := new(bytes.Buffer)
	a := &nntp.Article{
		Header: map[string][]string{},
	}

	a.Header["From"] = []string{data.poster}
	a.Header["Newsgroups"] = []string{data.group}
	a.Header["Message-ID"] = []string{"<" + data.msgId + ">"}
	a.Header["X-Newsposter"] = []string{"UsenetDrive"}

	subj := fmt.Sprintf(
		"[%d/%d] - \"%s\" yEnc (%d/%d)",
		data.fileNum,
		data.fileTotal,
		data.fileName,
		data.partNum,
		data.partTotal,
	)
	a.Header["Subject"] = []string{subj}

	// yEnc begin line
	buf.WriteString(fmt.Sprintf(
		"=ybegin part=%d total=%d line=128 size=%d name=%s\r\n",
		data.partNum,
		data.partTotal,
		data.fileSize,
		data.fileName,
	))
	// yEnc part line
	buf.WriteString(fmt.Sprintf("=ypart begin=%d end=%d\r\n", data.partBegin+1, data.partEnd))

	// Encoded data
	yenc.Encode(p, buf)
	// yEnc end line
	h := crc32.NewIEEE()
	h.Write(p)
	buf.WriteString(fmt.Sprintf("=yend size=%d part=%d pcrc32=%08X\r\n", data.partSize, data.partNum, h.Sum32()))
	a.Body = buf

	return a
}
