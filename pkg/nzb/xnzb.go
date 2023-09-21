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
	xnzb := new(xNzb)
	xnzb.XMLns = "http://www.newzbin.com/DTD/2003/nzb"
	xnzb.Head = make([]xNzbMeta, 0)
	for k, v := range nzb.Meta {
		xnzb.Head = append(xnzb.Head, xNzbMeta{Type: k, Value: v})
	}
	xnzb.File = make([]xNzbFile, 0)
	for _, f := range nzb.Files {
		xnzb.File = append(xnzb.File, xNzbFile{
			Poster:   f.Poster,
			Date:     f.Date,
			Subject:  f.Subject,
			Groups:   f.Groups,
			Segments: make([]xNzbSegment, 0),
		})
		for _, s := range f.Segments {
			xnzb.File[len(xnzb.File)-1].Segments = append(xnzb.File[len(xnzb.File)-1].Segments, xNzbSegment{
				Bytes:     s.Bytes,
				Number:    s.Number,
				MessageId: s.Id,
			})
		}
	}
	return xnzb
}

func xNzbFileToNzbFile(x *xNzbFile) NzbFile {
	nzbFile := NzbFile{
		Poster:   x.Poster,
		Date:     x.Date,
		Subject:  x.Subject,
		Groups:   x.Groups,
		Segments: make(NzbSegmentSlice, 0),
	}
	for i, _ := range x.Segments {
		nzbFile.Segments = append(nzbFile.Segments, xNzbSegmentToNzbSegment(&x.Segments[i]))
	}
	return nzbFile
}

func xNzbSegmentToNzbSegment(x *xNzbSegment) NzbSegment {
	return NzbSegment{
		Bytes:  x.Bytes,
		Number: x.Number,
		Id:     x.MessageId,
	}
}
