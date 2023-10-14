package nzbloader

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestNzbLoader_LoadFromFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := osfs.NewMockFileSystem(ctrl)
	cNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	nzbParserMock := nzb.NewMockNzbParser(ctrl)
	cache, err := lru.New[string, *NzbCache](10)
	assert.NoError(t, err)

	loader := &nzbLoader{
		cache:     cache,
		cNzb:      cNzb,
		fs:        fs,
		nzbParser: nzbParserMock,
	}

	assert.NoError(t, err)

	// Test case where the file is not found
	t.Run("file not found", func(t *testing.T) {
		fs.EXPECT().Open("file1.nzb").Return(nil, os.ErrNotExist)

		_, err = loader.LoadFromFile("file1.nzb")
		assert.Error(t, err)
	})

	// Test case where the file is found but the nzb is corrupted
	t.Run("corrupted nzb xml", func(t *testing.T) {
		mockFile := osfs.NewMockFile(ctrl)
		nzbParserMock.EXPECT().Parse(mockFile).Return(nil, errors.New("corrupted nzb xml"))

		fs.EXPECT().Open("file2.nzb").Return(mockFile, nil)
		cNzb.EXPECT().Add(gomock.Any(), "file2.nzb", "corrupted nzb xml").Return(nil)

		_, err = loader.LoadFromFile("file2.nzb")
		assert.Error(t, err)
	})

	t.Run("corrupted nzb metadata", func(t *testing.T) {
		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		nzbCopy := nzb

		delete(nzbCopy.Meta, "file_extension")

		mockFile := osfs.NewMockFile(ctrl)
		nzbParserMock.EXPECT().Parse(mockFile).Return(nzbCopy, nil)

		fs.EXPECT().Open("file2.nzb").Return(mockFile, nil)
		cNzb.EXPECT().Add(gomock.Any(), "file2.nzb", "corrupted nzb file, missing required metadata").Return(nil)

		_, err = loader.LoadFromFile("file2.nzb")
		assert.Error(t, err)
	})

	t.Run("error adding to corrupted list", func(t *testing.T) {
		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		nzbCopy := nzb

		delete(nzbCopy.Meta, "file_extension")

		mockFile := osfs.NewMockFile(ctrl)
		nzbParserMock.EXPECT().Parse(mockFile).Return(nzbCopy, nil)

		fs.EXPECT().Open("file2.nzb").Return(mockFile, nil)
		cNzb.EXPECT().Add(gomock.Any(), "file2.nzb", "corrupted nzb file, missing required metadata").
			Return(errors.New("error adding to corrupted list"))

		_, err = loader.LoadFromFile("file2.nzb")
		assert.Error(t, err)
		assert.Equal(t, "error adding to corrupted list", err.Error())
	})

	// Test case where the file is found and the nzb is not corrupted
	t.Run("not corrupted nzb", func(t *testing.T) {
		t.Cleanup(func() {
			loader.cache.Purge()
		})

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		mockFile := osfs.NewMockFile(ctrl)
		nzbParserMock.EXPECT().Parse(mockFile).Return(nzb, nil)

		fs.EXPECT().Open("file3.nzb").Return(mockFile, nil)
		item, err := loader.LoadFromFile("file3.nzb")
		assert.NoError(t, err)
		assert.Equal(t, nzb, item.Nzb)
	})

	// Test case where the file is found and the nzb is not corrupted and is already in the cache
	t.Run("not corrupted nzb in cache", func(t *testing.T) {
		t.Cleanup(func() {
			loader.cache.Purge()
		})

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		loader.cache.Add("file4.nzb", &NzbCache{
			Nzb: nzb,
			Metadata: usenet.Metadata{
				FileName:      "file4.mkv",
				FileExtension: "mkv",
				FileSize:      100,
				ChunkSize:     10,
				ModTime:       time.Now(),
			},
		})

		item, err := loader.LoadFromFile("file4.nzb")
		assert.NoError(t, err)
		assert.Equal(t, nzb, item.Nzb)
	})
}

func TestNzbLoader_EvictFromCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := osfs.NewMockFileSystem(ctrl)
	cNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	nzbParserMock := nzb.NewMockNzbParser(ctrl)
	cache, err := lru.New[string, *NzbCache](10)
	assert.NoError(t, err)

	loader := &nzbLoader{
		cache:     cache,
		cNzb:      cNzb,
		fs:        fs,
		nzbParser: nzbParserMock,
	}

	// Test case where the file is not in the cache
	t.Run("file not in cache", func(t *testing.T) {
		assert.False(t, loader.EvictFromCache("file1.nzb"))
	})

	// Test case where the file is in the cache
	t.Run("file in cache", func(t *testing.T) {
		t.Cleanup(func() {
			loader.cache.Purge()
		})

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		loader.cache.Add("file4.nzb", &NzbCache{
			Nzb: nzb,
			Metadata: usenet.Metadata{
				FileName:      "file4.mkv",
				FileExtension: "mkv",
				FileSize:      100,
				ChunkSize:     10,
				ModTime:       time.Now(),
			},
		})

		assert.True(t, loader.EvictFromCache("file4.nzb"))
		assert.Equal(t, 0, loader.cache.Len())
	})
}

func TestNzbLoader_RefreshCachedNzb(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fs := osfs.NewMockFileSystem(ctrl)
	cNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)
	nzbParserMock := nzb.NewMockNzbParser(ctrl)
	cache, err := lru.New[string, *NzbCache](10)
	assert.NoError(t, err)

	loader := &nzbLoader{
		cache:     cache,
		cNzb:      cNzb,
		fs:        fs,
		nzbParser: nzbParserMock,
	}

	// Test case where the nzb is corrupted
	t.Run("corrupted nzb", func(t *testing.T) {
		_, err = loader.RefreshCachedNzb("file1.nzb", &nzb.Nzb{})
		assert.Error(t, err)
	})

	// Test case where the nzb is not corrupted
	t.Run("not corrupted nzb", func(t *testing.T) {
		t.Cleanup(func() {
			loader.cache.Purge()
		})

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		loader.cache.Add("file4.nzb", &NzbCache{
			Nzb: nzb,
			Metadata: usenet.Metadata{
				FileName:      "file4.mkv",
				FileExtension: "mkv",
				FileSize:      100,
				ChunkSize:     10,
				ModTime:       time.Now(),
			},
		})

		ok, err := loader.RefreshCachedNzb("file4.nzb", nzb)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, 1, loader.cache.Len())
	})
}
