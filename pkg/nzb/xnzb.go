package nzb

import (
	"encoding/xml"
)

type xNzb struct {
	XMLName xml.Name   `xml:"nzb"`
	XMLns   string     `xml:"xmlns,attr"`
	Head    []xNzbMeta `xml:"head>meta"`
	File    []xNzbFile `xml:"file"`
}

type xNzbMeta struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

type xNzbFile struct {
	Poster   string        `xml:"poster,attr"`
	Date     int64         `xml:"date,attr"`
	Subject  string        `xml:"subject,attr"`
	Groups   []string      `xml:"groups>group"`
	Segments []xNzbSegment `xml:"segments>segment"`
}

type xNzbSegment struct {
	XMLName   xml.Name `xml:"segment"`
	Bytes     int64    `xml:"bytes,attr"`
	Number    int64    `xml:"number,attr"`
	MessageId string   `xml:",innerxml"`
}

func nzbToXNzb(nzb *Nzb) *xNzb {
	head := make([]xNzbMeta, 0)
	for k, v := range nzb.Meta {
		head = append(head, xNzbMeta{Type: k, Value: v})
	}
	file := make([]xNzbFile, len(nzb.Files))
	for i, f := range nzb.Files {
		segments := make([]xNzbSegment, len(f.Segments))
		for j, s := range f.Segments {
			segments[j] = xNzbSegment{
				Bytes:     s.Bytes,
				Number:    s.Number,
				MessageId: s.Id,
			}
		}

		file[i] = xNzbFile{
			Poster:   f.Poster,
			Date:     f.Date,
			Subject:  f.Subject,
			Groups:   f.Groups,
			Segments: segments,
		}
	}

	return &xNzb{
		XMLns: "http://www.newzbin.com/DTD/2003/nzb",
		Head:  head,
		File:  file,
	}
}

func xNzbFileToNzbFile(f *xNzbFile) NzbFile {
	segments := make(NzbSegmentSlice, len(f.Segments))
	for i, segment := range f.Segments {
		segments[i] = xNzbSegmentToNzbSegment(&segment)
	}

	return NzbFile{
		Poster:   f.Poster,
		Date:     f.Date,
		Subject:  f.Subject,
		Groups:   f.Groups,
		Segments: segments,
	}
}

func xNzbSegmentToNzbSegment(x *xNzbSegment) NzbSegment {
	return NzbSegment{
		Bytes:  x.Bytes,
		Number: x.Number,
		Id:     x.MessageId,
	}
}
