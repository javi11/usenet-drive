package filewriter

import (
	"bytes"
	"fmt"
	"hash/crc32"

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

func ArticleToBytes(p []byte, data *ArticleData, encoder *rapidyenc.Encoder) (*bytes.Buffer, error) {
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
	_, err := buf.WriteString(fmt.Sprintf("Subject: %s\r\n\r\n", subj))
	if err != nil {
		return nil, err
	}

	// yEnc begin line
	_, err = buf.WriteString(fmt.Sprintf(
		"=ybegin part=%d total=%d line=128 size=%d name=%s\r\n",
		data.partNum,
		data.partTotal,
		data.fileSize,
		data.fileName,
	))
	if err != nil {
		return nil, err
	}
	// yEnc part line
	_, err = buf.WriteString(fmt.Sprintf("=ypart begin=%d end=%d\r\n", data.partBegin+1, data.partEnd))
	if err != nil {
		return nil, err
	}

	// Encoded data
	encoded := encoder.Encode(p)
	// Write the actual data
	_, err = buf.Write(encoded)
	if err != nil {
		return nil, err
	}

	// Rapidyenc do not add \r\n to the end of the encoded data
	_, err = buf.WriteString("\r\n")
	if err != nil {
		return nil, err
	}

	// yEnc end line
	h := crc32.NewIEEE()
	_, err = h.Write(p)
	if err != nil {
		return nil, err
	}

	// Write the yEnc end line
	_, err = buf.WriteString(fmt.Sprintf("=yend size=%d part=%d pcrc32=%08X\r\n", data.partSize, data.partNum, h.Sum32()))
	if err != nil {
		return nil, err
	}

	return buf, nil
}
