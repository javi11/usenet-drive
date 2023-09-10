package uploader

import (
	"bytes"
	_ "embed"
	"encoding/xml"

	"github.com/chrisfarms/nzb"
	"github.com/javi11/usenet-drive/internal/usenet"
)

//go:embed test.nzb
var testNzb []byte

const (
	NzbHeader  = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	NzbDoctype = `<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">` + "\n"
)

type Nzb struct {
	XMLName xml.Name  `xml:"nzb"`
	XMLns   string    `xml:"xmlns,attr"`
	Head    []Meta    `xml:"head>meta"`
	File    []NzbFile `xml:"file"`
}

type Meta struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

type NzbFile struct {
	Poster   string       `xml:"poster,attr"`
	Date     int64        `xml:"date,attr"`
	Subject  string       `xml:"subject,attr"`
	Groups   []string     `xml:"groups>group"`
	Segments []NzbSegment `xml:"segments>segment"`
}

type NzbSegment struct {
	XMLName   xml.Name `xml:"segment"`
	Bytes     int64    `xml:"bytes,attr"`
	Number    int64    `xml:"number,attr"`
	MessageId string   `xml:",innerxml"`
}

func generateFakeNzb(fileName, fileExtension string) ([]byte, error) {
	reader := bytes.NewReader(testNzb)
	myNzb, err := nzb.New(reader)
	if err != nil {
		return nil, err
	}

	n := usenet.UpdateNzbMetadada(myNzb, usenet.UpdateableMetadata{
		FileName:      fileName,
		FileExtension: fileExtension,
	})

	return usenet.NzbToBytes(n)
}
