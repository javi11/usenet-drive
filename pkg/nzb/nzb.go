package nzb

//go:generate mockgen -source=./nzb.go -destination=./nzb_mock.go -package=nzb NzbParser

import (
	"bytes"
	"encoding/xml"
	"io"
)

type NzbParser interface {
	Parse(buf io.Reader) (*Nzb, error)
	ParseFromString(data string) (*Nzb, error)
}

type nzbParser struct{}

func NewNzbParser() NzbParser {
	return &nzbParser{}
}

func (p *nzbParser) ParseFromString(data string) (*Nzb, error) {
	return p.Parse(bytes.NewBufferString(data))
}

func (p *nzbParser) Parse(buf io.Reader) (*Nzb, error) {
	xnzb := &xNzb{}
	err := xml.NewDecoder(buf).Decode(xnzb)
	if err != nil {
		return nil, err
	}
	// convert to nicer format
	nzb := &Nzb{}
	// convert metadata
	nzb.Meta = make(map[string]string)
	for _, md := range xnzb.Head {
		nzb.Meta[md.Type] = md.Value
	}

	nzb.Files = make([]*NzbFile, len(xnzb.File))
	for i, file := range xnzb.File {
		nzb.Files[i] = xNzbFileToNzbFile(file)
	}
	return nzb, nil
}

const (
	NzbHeader  = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	NzbDoctype = `<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">` + "\n"
)

type Nzb struct {
	Meta  map[string]string
	Files []*NzbFile
}

type NzbFile struct {
	Groups   []string      `xml:"groups>group"`
	Segments []*NzbSegment `xml:"segments>segment"`
	Poster   string        `xml:"poster,attr"`
	Date     int64         `xml:"date,attr"`
	Subject  string        `xml:"subject,attr"`
	Part     int64
}

type NzbSegment struct {
	XMLName xml.Name `xml:"segment"`
	Bytes   int64    `xml:"bytes,attr"`
	Number  int64    `xml:"number,attr"`
	Id      string   `xml:",innerxml"`
}

type UpdateableMetadata struct {
	FileName      string
	FileExtension string
}

func (n *Nzb) UpdateMetadata(metadata UpdateableMetadata) *Nzb {
	if metadata.FileName != "" {
		n.Meta["file_name"] = metadata.FileName
	}
	if metadata.FileExtension != "" {
		n.Meta["file_extension"] = metadata.FileExtension
	}

	return n

}

func (n *Nzb) ToBytes() ([]byte, error) {
	xNzb := nzbToXNzb(n)

	output, err := xml.MarshalIndent(xNzb, "", "  ")
	if err != nil {
		return nil, err
	}
	output = []byte(NzbHeader + NzbDoctype + string(output))

	return output, nil
}
