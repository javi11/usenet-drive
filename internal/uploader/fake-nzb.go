package uploader

import (
	"bytes"
	_ "embed"

	"github.com/javi11/usenet-drive/pkg/nzb"
)

//go:embed test.nzb
var testNzb []byte

func generateFakeNzb(fileName, fileExtension string) ([]byte, error) {
	reader := bytes.NewReader(testNzb)
	myNzb, err := nzb.NzbFromBuffer(reader)
	if err != nil {
		return nil, err
	}

	n := myNzb.UpdateMetadada(nzb.UpdateableMetadata{
		FileName:      fileName,
		FileExtension: fileExtension,
	})

	return n.ToBytes()
}
