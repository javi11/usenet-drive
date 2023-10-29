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
	Next(from int) (*nzb.NzbSegment, bool)
}

type NzbReader interface {
	GetMetadata() (usenet.Metadata, error)
	GetGroups() ([]string, error)
	GetSegment(segmentIndex int) (*nzb.NzbSegment, bool)
}

type nzbReader struct {
	decoder  *xml.Decoder
	metadata *usenet.Metadata
	groups   []string
	segments map[int64]*nzb.NzbSegment
	mx       sync.RWMutex
}

func NewNzbReader(reader io.Reader) NzbReader {
	return &nzbReader{
		decoder:  xml.NewDecoder(reader),
		segments: map[int64]*nzb.NzbSegment{},
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

func (r *nzbReader) GetSegment(segmentIndex int) (*nzb.NzbSegment, bool) {
	r.mx.RLock()
	defer r.mx.RLocker()

	segmentNumber := int64(segmentIndex + 1)
	// Check if the segment is already in the cache
	if r.segments[segmentNumber] != nil {
		return r.segments[segmentNumber], true
	}

	// Check if there are more segments to read from the XML stream
	for {
		token, err := r.decoder.Token()
		if err != nil {
			return nil, false
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == "segment" {
			// Read the next segment from the XML stream
			var segment nzb.NzbSegment
			r.decoder.DecodeElement(&segment, &se)
			r.segments[segmentNumber] = &segment

			if segment.Number == segmentNumber {
				return &segment, true
			}

			continue
		}
	}

}
