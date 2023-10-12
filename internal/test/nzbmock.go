package test

import (
	"bytes"
	_ "embed"

	"github.com/javi11/usenet-drive/pkg/nzb"
)

//go:embed nzbmock.xml
var nzbmock []byte

func NewNzbMock() (*nzb.Nzb, error) {
	buff := bytes.NewBuffer(nzbmock)
	return nzb.NzbFromBuffer(buff)
}
