package test

import (
	"bytes"
	_ "embed"

	"github.com/javi11/usenet-drive/pkg/nzb"
)

//go:embed nzbmock.xml
var nzbmock []byte

func NewNzbMock() (*nzb.Nzb, error) {
	nzbParser := nzb.NewNzbParser()

	buff := bytes.NewBuffer(nzbmock)
	return nzbParser.Parse(buff)
}
