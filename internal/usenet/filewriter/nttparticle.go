package filewriter

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/mnightingale/rapidyenc"
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

func ArticleToReader(p []byte, data ArticleData, encoder *rapidyenc.Encoder) (io.Reader, error) {
	subj := fmt.Sprintf(
		"[%d/%d] - \"%s\" yEnc (%d/%d)",
		data.fileNum,
		data.fileTotal,
		data.fileName,
		data.partNum,
		data.partTotal,
	)
	header := fmt.Sprintf("From: %s\r\nNewsgroups: %s\r\nMessage-ID: <%s>\r\nX-Newsposter: UsenetDrive\r\nSubject: %s\r\n\r\n=ybegin part=%d total=%d line=128 size=%d name=%s\r\n=ypart begin=%d end=%d\r\n",
		data.poster,
		data.group,
		data.msgId,
		subj,
		data.partNum,
		data.partTotal,
		data.fileSize,
		data.fileName,
		data.partBegin+1,
		data.partEnd,
	)

	// Encoded data
	encoded := encoder.Encode(p)

	// yEnc end line
	h := crc32.NewIEEE()
	_, err := h.Write(p)
	if err != nil {
		return nil, err
	}
	footer := fmt.Sprintf("\r\n=yend size=%d part=%d pcrc32=%08X\r\n", data.partSize, data.partNum, h.Sum32())

	size := len(header) + len(encoded) + len(footer)
	buf := bytes.NewBuffer(make([]byte, 0, size))

	_, err = buf.WriteString(header)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(encoded)
	if err != nil {
		return nil, err
	}
	_, err = buf.WriteString(footer)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
