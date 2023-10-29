package nzbloader

import (
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

type NzbWriter interface {
	UpdateMetadata(filePath string, metadata nzb.UpdateableMetadata) error
}

type nzbWriter struct {
	fs        osfs.FileSystem
	nzbParser nzb.NzbParser
}

func NewNzbWriter(fs osfs.FileSystem, nzbParser nzb.NzbParser) NzbWriter {
	return &nzbWriter{
		fs:        fs,
		nzbParser: nzbParser,
	}
}

func (nw *nzbWriter) UpdateMetadata(filePath string, metadata nzb.UpdateableMetadata) error {
	f, err := nw.fs.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	nzbFile, err := nw.nzbParser.Parse(f)
	if err != nil {
		return err
	}

	newNzb := nzbFile.UpdateMetadata(metadata)

	b, err := newNzb.ToBytes()
	if err != nil {
		return err
	}

	err = nw.fs.WriteFile(filePath, b, 0766)
	if err != nil {
		return err
	}

	return nil
}
