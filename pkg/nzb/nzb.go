package nzb

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"sort"
)

const (
	NzbHeader  = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	NzbDoctype = `<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">` + "\n"
)

type NzbFileSlice []NzbFile

func (s NzbFileSlice) Len() int           { return len(s) }
func (s NzbFileSlice) Less(i, j int) bool { return s[i].Part < s[j].Part }
func (s NzbFileSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type NzbSegmentSlice []NzbSegment

func (s NzbSegmentSlice) Len() int           { return len(s) }
func (s NzbSegmentSlice) Less(i, j int) bool { return s[i].Number < s[j].Number }
func (s NzbSegmentSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type Nzb struct {
	Meta  map[string]string
	Files NzbFileSlice
}

type NzbFile struct {
	Groups   []string        `xml:"groups>group"`
	Segments NzbSegmentSlice `xml:"segments>segment"`
	Poster   string          `xml:"poster,attr"`
	Date     int64           `xml:"date,attr"`
	Subject  string          `xml:"subject,attr"`
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

func (n *Nzb) WriteIntoFile(f *os.File) error {
	sort.Sort(n.Files)
	for i, _ := range n.Files {
		sort.Sort(n.Files[i].Segments)
	}
	nzb := nzbToXNzb(n)
	if output, err := xml.MarshalIndent(nzb, "", "    "); err == nil {
		output = []byte(NzbHeader + NzbDoctype + string(output))
		_, err := f.Write(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Nzb) UpdateMetadada(metadata UpdateableMetadata) *Nzb {
	if metadata.FileName != "" {
		n.Meta["file_name"] = metadata.FileName
	}
	if metadata.FileExtension != "" {
		n.Meta["file_extension"] = metadata.FileExtension
	}

	return n

}

func (n *Nzb) ToBytes() ([]byte, error) {
	sort.Sort(n.Files)
	for i, _ := range n.Files {
		sort.Sort(n.Files[i].Segments)
	}
	xNzb := nzbToXNzb(n)

	output, err := xml.MarshalIndent(xNzb, "", "    ")
	if err != nil {
		return nil, err
	}
	output = []byte(NzbHeader + NzbDoctype + string(output))

	return output, nil
}

func NzbFromString(data string) (*Nzb, error) {
	return NzbFromBuffer(bytes.NewBufferString(data))
}

func NzbFromBuffer(buf io.Reader) (*Nzb, error) {
	xnzb := new(xNzb)
	err := xml.NewDecoder(buf).Decode(xnzb)
	if err != nil {
		return nil, err
	}
	// convert to nicer format
	nzb := new(Nzb)
	// convert metadata
	nzb.Meta = make(map[string]string)
	for _, md := range xnzb.Head {
		nzb.Meta[md.Type] = md.Value
	}
	// copy files into (sortable) NzbFileSlice
	nzb.Files = make(NzbFileSlice, 0)
	for i, _ := range xnzb.File {
		nzb.Files = append(nzb.Files, xNzbFileToNzbFile(&xnzb.File[i]))
	}
	return nzb, nil
}
