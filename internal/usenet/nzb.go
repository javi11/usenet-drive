package usenet

import (
	_ "embed"
	"encoding/xml"

	"github.com/chrisfarms/nzb"
)

const (
	NzbHeader  = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	NzbDoctype = `<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">` + "\n"
)

type xNzb struct {
	XMLName xml.Name  `xml:"nzb"`
	XMLns   string    `xml:"xmlns,attr"`
	Head    []meta    `xml:"head>meta"`
	File    []nzbFile `xml:"file"`
}

type meta struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

type nzbFile struct {
	Poster   string       `xml:"poster,attr"`
	Date     int64        `xml:"date,attr"`
	Subject  string       `xml:"subject,attr"`
	Groups   []string     `xml:"groups>group"`
	Segments []nzbSegment `xml:"segments>segment"`
}

type nzbSegment struct {
	XMLName   xml.Name `xml:"segment"`
	Bytes     int64    `xml:"bytes,attr"`
	Number    int64    `xml:"number,attr"`
	MessageId string   `xml:",innerxml"`
}

type UpdateableMetadata struct {
	FileName      string
	FileExtension string
}

func UpdateNzbMetadada(myNzb *nzb.Nzb, metadata UpdateableMetadata) *nzb.Nzb {
	if metadata.FileName != "" {
		myNzb.Meta["file_name"] = metadata.FileName
	}
	if metadata.FileExtension != "" {
		myNzb.Meta["file_extension"] = metadata.FileExtension
	}

	return myNzb

}

func NzbToBytes(myNzb *nzb.Nzb) ([]byte, error) {
	xmlNzb := xNzb{
		XMLName: xml.Name{Local: "nzb", Space: "http://www.newzbin.com/DTD/2003/nzb"},
		XMLns:   "http://www.newzbin.com/DTD/2003/nzb",
		Head:    make([]meta, 0),
		File:    make([]nzbFile, 0),
	}

	for key, value := range myNzb.Meta {
		xmlNzb.Head = append(xmlNzb.Head, meta{
			Type:  key,
			Value: value,
		})

	}

	for _, file := range myNzb.Files {
		f := nzbFile{
			Poster:   file.Poster,
			Date:     int64(file.Date),
			Subject:  file.Subject,
			Groups:   file.Groups,
			Segments: []nzbSegment{},
		}

		for _, segment := range file.Segments {
			f.Segments = append(f.Segments, nzbSegment{
				Bytes:     int64(segment.Bytes),
				Number:    int64(segment.Number),
				MessageId: segment.Id,
			})
		}

		xmlNzb.File = append(xmlNzb.File, f)

	}

	output, err := xml.MarshalIndent(xmlNzb, "", "    ")
	if err != nil {
		return nil, err
	}
	output = []byte(NzbHeader + NzbDoctype + string(output))

	return output, nil
}
