package filewriter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArticleToBytes(t *testing.T) {
	data := &ArticleData{
		poster:    "test@example.com",
		group:     "alt.binaries.test",
		msgId:     "1234567890",
		fileNum:   1,
		fileTotal: 1,
		fileName:  "testfile.txt",
		partNum:   1,
		partTotal: 1,
		fileSize:  10,
		partBegin: 0,
		partEnd:   9,
		partSize:  10,
	}

	p := []byte("test data1")

	resBytes, err := ArticleToBytes(p, data)
	assert.NoError(t, err)

	assert.Equal(t, "From: test@example.com\r\nNewsgroups: alt.binaries.test\r\nMessage-ID: <1234567890>\r\nX-Newsposter: UsenetDrive\r\nSubject: [1/1] - \"testfile.txt\" yEnc (1/1)\r\n\r\n=ybegin part=1 total=1 line=128 size=10 name=testfile.txt\r\n=ypart begin=1 end=9\r\n\x9e\x8f\x9d\x9eJ\x8e\x8b\x9e\x8b[\r\n=yend size=10 part=1 pcrc32=A66035B9\r\n", string(resBytes))
}
