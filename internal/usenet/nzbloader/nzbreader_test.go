package nzbloader

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/javi11/usenet-drive/pkg/mmap"
)

var segments = []struct {
	input int
}{
	{input: 0},
	{input: 2212},
	{input: 5415},
	{input: 14719},
	{input: 42957},
}

func BenchmarkGetSegmentWithMMapFile(b *testing.B) {
	f, err := os.Open("../../test/big.nzb")
	if err != nil {
		b.Fatal(err)
	}

	fs, err := f.Stat()
	if err != nil {
		b.Fatal(err)
	}

	mmapFile, err := mmap.MmapFileWithSize(f, int(fs.Size()))
	if err != nil {
		b.Fatal(err)
	}
	defer mmapFile.Close()

	reader := NewNzbReader(bytes.NewReader(mmapFile.Bytes()))
	defer reader.Close()

	_, err = reader.GetMetadata()
	if err != nil {
		b.Fatal(err)
	}

	_, err = reader.GetGroups()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for _, v := range segments {
		b.Run(fmt.Sprintf("segment_index_%d", v.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// get last segment
				_, has := reader.GetSegment(v.input)
				if !has {
					b.Fatal("segment not found")
				}
			}
		})
	}
}

func BenchmarkGetSegmentWithoutMMap(b *testing.B) {
	f, err := os.Open("../../test/big.nzb")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	reader := NewNzbReader(f)
	defer reader.Close()

	_, err = reader.GetMetadata()
	if err != nil {
		b.Fatal(err)
	}

	_, err = reader.GetGroups()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for _, v := range segments {
		b.Run(fmt.Sprintf("segment_index_%d", v.input), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// get last segment
				_, has := reader.GetSegment(v.input)
				if !has {
					b.Fatal("segment not found")
				}
			}
		})
	}
}
