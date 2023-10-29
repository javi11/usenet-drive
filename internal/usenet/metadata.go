package usenet

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Metadata struct {
	FileName      string    `json:"file_name"`
	FileExtension string    `json:"file_extension"`
	FileSize      int64     `json:"file_size"`
	ModTime       time.Time `json:"mod_time"`
	ChunkSize     int64     `json:"chunk_size"`
}

func LoadMetadataFromMap(metadata map[string]string) (Metadata, error) {
	if metadata["file_name"] == "" ||
		metadata["file_size"] == "" ||
		metadata["mod_time"] == "" ||
		metadata["file_extension"] == "" ||
		metadata["subject"] == "" {
		return Metadata{}, fmt.Errorf("corrupted nzb file, missing required metadata")
	}

	fileSize, err := strconv.ParseInt(metadata["file_size"], 10, 64)
	if err != nil {
		return Metadata{}, err
	}

	chunkSize := int64(0)

	cz := metadata["chunk_size"]
	if cz != "" {
		chunkSize, err = strconv.ParseInt(cz, 10, 64)
		if err != nil {
			return Metadata{}, err
		}
	} else {
		// Fallback to old subject format
		// Chunk size is present in the file subject string
		// segment size is not the real size of a segment. Segment size = chunk size + yenc overhead
		chunkSize, err = getChunkSizeFromSubject(metadata["subject"])
		if err != nil {
			return Metadata{}, fmt.Errorf("corrupted nzb file, no files found")
		}
	}

	modTime, err := time.Parse(time.DateTime, metadata["mod_time"])
	if err != nil {
		return Metadata{}, err
	}

	if metadata["file_extension"] == "" {
		return Metadata{}, fmt.Errorf("corrupted nzb file, file extension not found")
	}

	return Metadata{
		FileName:      metadata["file_name"],
		FileExtension: metadata["file_extension"],
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
