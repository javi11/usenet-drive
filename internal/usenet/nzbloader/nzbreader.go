package nzbloader

//go:generate mockgen -source=./nzbreader.go -destination=./nzbreader_mock.go -package=nzbloader NzbReader

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/pkg/nzb"
)

type NzbReader interface {
	GetMetadata() (usenet.Metadata, error)
	GetGroups() ([]string, error)
	GetSegment(segmentIndex int) (nzb.NzbSegment, bool)
	Close()
}

type nzbReader struct {
	decoder        *xml.Decoder
	metadata       usenet.Metadata
	groups         []string
	segments       map[int64]nzb.NzbSegment
	close          chan struct{}
	currentSegment atomic.Int64
}

func NewNzbReader(reader io.Reader) NzbReader {
	return &nzbReader{
		decoder:  xml.NewDecoder(reader),
		close:    make(chan struct{}),
		segments: map[int64]nzb.NzbSegment{},
	}
}

func (r *nzbReader) Close() {
	close(r.close)
	clear(r.segments)
	r.segments = nil
	clear(r.groups)
	r.groups = nil
}

func (r *nzbReader) GetMetadata() (usenet.Metadata, error) {
	if r.metadata.ChunkSize != 0 {
		return r.metadata, nil
	}

	metadata := map[string]string{}
	for {
		select {
		case <-r.close:
			return usenet.Metadata{}, fmt.Errorf("nzb file closed")
		default:
			token, err := r.decoder.RawToken()
			if err != nil {
				if err == io.EOF {
					break
				}
				return usenet.Metadata{}, err
			}

			switch se := token.(type) {
			case xml.StartElement:
				if se.Name.Local == "meta" {
					var key string
					for _, attr := range se.Attr {
						if attr.Name.Local == "type" {
							key = attr.Value
						}
					}
					if key == "" {
						return usenet.Metadata{}, fmt.Errorf("missing type attribute in meta element")
					}
					t, err := r.decoder.RawToken()
					if err != nil {
						return usenet.Metadata{}, err
					}
					metadata[key] = string(t.(xml.CharData))
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
					r.metadata = m

					return m, nil
				}
			}
		}
	}
}

func (r *nzbReader) GetGroups() ([]string, error) {
	if r.metadata.ChunkSize == 0 {
		// we need to maintain the order of the calls to GetMetadata and GetGroups
		if _, err := r.GetMetadata(); err != nil {
			return nil, err
		}
	}

	if r.groups != nil {
		return r.groups, nil
	}

	for {
		select {
		case <-r.close:
			return nil, fmt.Errorf("nzb file closed")
		default:
			token, err := r.decoder.RawToken()
			if err != nil {
				if err == io.EOF {
					if len(r.groups) == 0 {
						return nil, fmt.Errorf("corrupted nzb file, missing groups element in nzb file")
					}

					return r.groups, nil
				}
				return nil, err
			}

			switch se := token.(type) {
			case xml.StartElement:
				if se.Name.Local == "groups" {
					for {
						token, err := r.decoder.RawToken()
						if err != nil {
							return nil, err
						}

						switch se := token.(type) {
						case xml.StartElement:
							if se.Name.Local == "group" {
								t, err := r.decoder.RawToken()
								if err != nil {
									return nil, err
								}
								group := string(t.(xml.CharData))

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
	}
}

func (r *nzbReader) GetSegment(segmentIndex int) (nzb.NzbSegment, bool) {
	// Check if the segment is already in the cache
	segmentNumber := int64(segmentIndex + 1)
	// Check if the segment is already in the cache
	if s, ok := r.segments[segmentNumber]; ok {
		return s, true
	}

	if r.groups == nil {
		_, err := r.GetGroups()
		if err != nil {
			return nzb.NzbSegment{}, false
		}
	}
	// Check if there are more segments to read from the XML stream
	for {
		select {
		case <-r.close:
			return nzb.NzbSegment{}, false
		default:
			token, err := r.decoder.RawToken()
			if err != nil {
				return nzb.NzbSegment{}, false
			}

			if se, ok := token.(xml.StartElement); ok && se.Name.Local == "segment" {
				// Read the next segment from the XML stream
				var segment nzb.NzbSegment
				err := segment.UnmarshalXML(r.decoder, se)
				if err != nil {
					return nzb.NzbSegment{}, false
				}

				r.segments[segment.Number] = segment

				if segment.Number == int64(segmentIndex+1) {
					return segment, true
				}
			}
		}
	}
}

func (r *nzbReader) NextSegment() (nzb.NzbSegment, bool) {
	segment, has := r.GetSegment(int(r.currentSegment.Load()))
	if !has {
		return nzb.NzbSegment{}, false
	}
	r.currentSegment.Add(1)
	return segment, true
}

func (r *nzbReader) ResetTo(segmentIndex int) {
	r.currentSegment.Store(int64(segmentIndex))
}
