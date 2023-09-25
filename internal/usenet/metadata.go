package usenet

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/javi11/usenet-drive/pkg/nzb"
)

type Metadata struct {
	FileName      string    `json:"file_name"`
	FileExtension string    `json:"file_extension"`
	FileSize      int64     `json:"file_size"`
	ModTime       time.Time `json:"mod_time"`
	ChunkSize     int64     `json:"chunk_size"`
}

func LoadMetadataFromNzb(nzbFile *nzb.Nzb) (Metadata, error) {
	if nzbFile.Meta["file_name"] == "" ||
		nzbFile.Meta["file_size"] == "" ||
		nzbFile.Meta["mod_time"] == "" ||
		nzbFile.Meta["file_extension"] == "" {
		return Metadata{}, fmt.Errorf("corrupted nzb file, missing required metadata")
	}

	fileSize, err := strconv.ParseInt(nzbFile.Meta["file_size"], 10, 64)
	if err != nil {
		return Metadata{}, err
	}

	if len(nzbFile.Files) == 0 {
		return Metadata{}, fmt.Errorf("corrupted nzb file, no files found")
	}

	chunkSize := int64(0)

	cz := nzbFile.Meta["chunk_size"]
	if cz != "" {
		chunkSize, err = strconv.ParseInt(cz, 10, 64)
		if err != nil {
			return Metadata{}, err
		}
	} else {
		// Fallback to old subject format
		// Chunk size is present in the file subject string
		// segment size is not the real size of a segment. Segment size = chunk size + yenc overhead
		chunkSize, err = getChunkSizeFromSubject(nzbFile.Files[0].Subject)
		if err != nil {
			return Metadata{}, fmt.Errorf("corrupted nzb file, no files found")
		}
	}

	modTime, err := time.Parse(time.DateTime, nzbFile.Meta["mod_time"])
	if err != nil {
		return Metadata{}, err
	}

	if nzbFile.Meta["file_extension"] == "" {
		return Metadata{}, fmt.Errorf("corrupted nzb file, file extension not found")
	}

	return Metadata{
		FileName:      nzbFile.Meta["file_name"],
		FileExtension: nzbFile.Meta["file_extension"],
		FileSize:      fileSize,
		ChunkSize:     chunkSize,
		ModTime:       modTime,
	}, nil
}

func getChunkSizeFromSubject(s string) (int64, error) {
	re := regexp.MustCompile(`size=(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no size found in string")
	}
	size, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}
