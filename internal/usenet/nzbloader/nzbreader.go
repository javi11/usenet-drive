package nzbloader

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type NzbIterator interface {
	Next() (*nzb.NzbSegment, bool)
	Seek(segmentIndex int)
}

type NzbReader interface {
	GetMetadata() (usenet.Metadata, error)
	GetGroups() ([]string, error)
	GetIterator() (NzbIterator, error)
}

type nzbReader struct {
	decoder  *xml.Decoder
	metadata *usenet.Metadata
	groups   []string
	mx       sync.Mutex
}

func NewNzbReader(reader io.Reader) NzbReader {
	return &nzbReader{
		decoder: xml.NewDecoder(reader),
	}
}

func (r *nzbReader) GetMetadata() (usenet.Metadata, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.metadata != nil {
		return *r.metadata, nil
	}

	metadata := map[string]string{}
	for {
		token, err := r.decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return usenet.Metadata{}, err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "meta" {
				var key, value string
				for _, attr := range se.Attr {
					if attr.Name.Local == "type" {
						key = attr.Value
					}
				}
				if key == "" {
					return usenet.Metadata{}, fmt.Errorf("missing type attribute in meta element")
				}
				if err := r.decoder.DecodeElement(&value, &se); err != nil {
					return usenet.Metadata{}, err
				}
				metadata[key] = value
			}
			if se.Name.Local == "file" {
				var value string
				for _, attr := range se.Attr {
					if attr.Name.Local == "subject" {
						value = attr.Value
					}
				}
				metadata["subject"] = value

				// We have all the metadata we need
				m, err := usenet.LoadMetadataFromMap(metadata)
				if err != nil {
					return usenet.Metadata{}, err
				}
				r.metadata = &m

				return m, nil
			}
		}
	}

	return usenet.Metadata{}, fmt.Errorf("corrupted nzb file, missing file element in nzb file")
}

func (r *nzbReader) GetGroups() ([]string, error) {
	if r.metadata == nil {
		// we need to maintain the order of the calls to GetMetadata and GetGroups
		if _, err := r.GetMetadata(); err != nil {
			return nil, err
		}
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	if r.groups != nil {
		return r.groups, nil
	}

	for {
		token, err := r.decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "groups" {
				for {
					token, err := r.decoder.Token()
					if err != nil {
						return nil, err
					}

					switch se := token.(type) {
					case xml.StartElement:
						if se.Name.Local == "group" {
							var group string
							if err := r.decoder.DecodeElement(&group, &se); err != nil {
								return nil, err
							}
							r.groups = append(r.groups, group)
						}
					case xml.EndElement:
						if se.Name.Local == "groups" {
							if len(r.groups) == 0 {
								return nil, fmt.Errorf("corrupted nzb file, missing groups element in nzb file")
							}

							return r.groups, nil
						}
					}
				}
			}
		}
	}

	if len(r.groups) == 0 {
		return nil, fmt.Errorf("corrupted nzb file, missing groups element in nzb file")
	}

	return r.groups, nil
}

func (r *nzbReader) GetIterator() (NzbIterator, error) {
	if len(r.groups) == 0 {
		// we need to maintain the order of the calls to GetMetadata, GetGroups and GetIterator
		if _, err := r.GetGroups(); err != nil {
			return nil, err
		}
	}

	return &nzbIterator{decoder: r.decoder}, nil
}

type nzbIterator struct {
	decoder      *xml.Decoder
	currentIndex int
	segments     []*nzb.NzbSegment
	mx           sync.Mutex
}

func (it *nzbIterator) Next() (*nzb.NzbSegment, bool) {
	it.mx.Lock()
	defer it.mx.Unlock()
	defer func() { it.currentIndex++ }()

	// Check if the segment is already in the cache
	if it.currentIndex+1 <= len(it.segments) && it.segments[it.currentIndex] != nil {
		return it.segments[it.currentIndex], true
	}

	// Check if there are more segments to read from the XML stream
	for {
		token, err := it.decoder.Token()
		if err != nil {
			return nil, false
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == "segment" {
			// Read the next segment from the XML stream
			var segment nzb.NzbSegment
			it.decoder.DecodeElement(&segment, nil)
			it.segments = append(it.segments, &segment)

			return &segment, true
		}
	}

}

func (it *nzbIterator) Seek(segmentIndex int) {
	it.mx.Lock()
	defer it.mx.Unlock()

	it.currentIndex = segmentIndex
	// Check if the segment is already in the cache
	if it.currentIndex+1 <= len(it.segments) && it.segments[it.currentIndex] != nil {
		return
	}

	// Seek to a specific segment in the XML stream
	for {
		token, err := it.decoder.Token()
		if err != nil {
			return
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == "segment" {
			it.currentIndex++
			var segment nzb.NzbSegment
			it.decoder.DecodeElement(&segment, &se)
			it.segments = append(it.segments, &segment)

			if segment.Number == int64(segmentIndex+1) {
				return
			}
		}
	}
}
