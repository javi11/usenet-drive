package filewriter

import (
	"bytes"
	"fmt"
	"hash/crc32"

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

func ArticleToBytes(p []byte, data *ArticleData) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	buf.WriteString(fmt.Sprintf("From: %s\r\n", data.poster))
	buf.WriteString(fmt.Sprintf("Newsgroups: %s\r\n", data.group))
	buf.WriteString(fmt.Sprintf("Message-ID: <%s>\r\n", data.msgId))
	buf.WriteString("X-Newsposter: UsenetDrive\r\n")

	// Build subject
	subj := fmt.Sprintf(
		"[%d/%d] - \"%s\" yEnc (%d/%d)",
		data.fileNum,
		data.fileTotal,
		data.fileName,
		data.partNum,
		data.partTotal,
	)
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n\r\n", subj))

	// yEnc begin line
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
	err := yenc.Encode(p, buf)
	if err != nil {
		return nil, err
	}
	// yEnc end line
	h := crc32.NewIEEE()
	h.Write(p)
	buf.WriteString(fmt.Sprintf("=yend size=%d part=%d pcrc32=%08X\r\n", data.partSize, data.partNum, h.Sum32()))

	return buf, nil
}
