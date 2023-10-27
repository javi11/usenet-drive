package filereader

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestNewFileInfoWithStat(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	nl := nzbloader.NewMockNzbLoader(ctrl)

	t.Run("Nzb file corrupted", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)

		nl.EXPECT().LoadFromFile("corrupted-nzb.nzb").Return(nil, ErrCorruptedNzb).Times(1)

		_, err := NewFileInfoWithStat("corrupted-nzb.nzb", log, nl, fstat)
		assert.Error(t, err)
		assert.Equal(t, ErrCorruptedNzb, err)
	})

	// Test case when file exists
	t.Run("File exists", func(t *testing.T) {
		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fstat := osfs.NewMockFileInfo(ctrl)

		fstat.EXPECT().Name().Return("test.nzb").Times(1)
		fstat.EXPECT().Mode().Return(os.FileMode(0)).Times(1)

		expectedTime := time.Now()

		nl.EXPECT().LoadFromFile("test.nzb").Return(&nzbloader.NzbCache{
			Nzb: nzb,
			Metadata: &usenet.Metadata{
				FileSize:      10,
				ModTime:       expectedTime,
				FileExtension: ".nzb",
				ChunkSize:     5,
			},
		}, nil).Times(1)

		info, err := NewFileInfoWithStat("test.nzb", log, nl, fstat)
		assert.NoError(t, err)
		assert.Equal(t, "test.nzb", info.Name())
		assert.Equal(t, int64(10), info.Size())
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0), info.Mode())
		assert.Equal(t, expectedTime, info.ModTime())

		os.Remove("test.nzb")
	})
}

func TestNeFileInfoWithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	metadata := &usenet.Metadata{
		FileSize:      100,
		ModTime:       time.Now(),
		FileExtension: ".nzb",
		ChunkSize:     5,
	}

	// Test case when file does not exist
	t.Run("Files does not exist", func(t *testing.T) {
		fs.EXPECT().Stat("non-existent-file.nzb").Return(nil, os.ErrNotExist)

		_, err := NeFileInfoWithMetadata("non-existent-file.nzb", metadata, fs)
		assert.Error(t, err)
		assert.Equal(t, os.ErrNotExist, err)
	})

	// Test case when file exists
	t.Run("File exists", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)

		fstat.EXPECT().Name().Return("test.nzb").Times(1)
		fstat.EXPECT().Mode().Return(os.FileMode(0)).Times(1)

		fs.EXPECT().Stat("test.nzb").Return(fstat, nil).Times(1)

		info, err := NeFileInfoWithMetadata("test.nzb", metadata, fs)
		assert.NoError(t, err)
		assert.Equal(t, "test.nzb", info.Name())
		assert.Equal(t, int64(100), info.Size())
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0), info.Mode())
		assert.Equal(t, metadata.ModTime, info.ModTime())

		os.Remove("test.nzb")
	})
}
