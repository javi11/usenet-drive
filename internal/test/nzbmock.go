package test

import (
	"bytes"
	_ "embed"

	"github.com/javi11/usenet-drive/pkg/nzb"
)

//go:embed nzbmock.xml
var NzbFile []byte

//go:embed corruptednzbmock.xml
var CorruptedNzbFile []byte

func NewNzbMock() (*nzb.Nzb, error) {
	return nzb.ParseFromBuffer(bytes.NewBuffer(NzbFile))
}
