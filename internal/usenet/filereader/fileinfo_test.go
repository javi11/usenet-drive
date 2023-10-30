package filereader

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestNewFileInfoWithStat(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()

	t.Run("Nzb file corrupted", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)
		fs := osfs.NewMockFileSystem(ctrl)

		f, err := os.Open("../../test/corruptednzbmock.xml")
		assert.NoError(t, err)
		fs.EXPECT().Open("corrupted-nzb.nzb").Return(f, nil).Times(1)

		_, err = NewFileInfoWithStat(fs, "corrupted-nzb.nzb", log, fstat)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrCorruptedNzb)
	})

	// Test case when file exists
	t.Run("File exists", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)
		fs := osfs.NewMockFileSystem(ctrl)

		fstat.EXPECT().Name().Return("test.mkv.nzb").Times(1)
		fstat.EXPECT().Mode().Return(os.FileMode(0)).Times(1)

		f, err := os.Open("../../test/nzbmock.xml")
		assert.NoError(t, err)
		fs.EXPECT().Open("test.mkv.nzb").Return(f, nil).Times(1)

		expectedTime, err := time.Parse(time.DateTime, "2023-09-22 20:06:09")
		assert.NoError(t, err)

		info, err := NewFileInfoWithStat(fs, "test.mkv.nzb", log, fstat)
		assert.NoError(t, err)
		assert.Equal(t, "test.mkv.bin", info.Name())
		assert.Equal(t, int64(1442682314), info.Size())
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0), info.Mode())
		assert.Equal(t, expectedTime, info.ModTime())
	})
}

func TestNeFileInfoWithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	metadata := usenet.Metadata{
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
