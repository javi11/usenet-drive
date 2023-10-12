package filereader

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/pkg/osfs"
)

func TestMax(t *testing.T) {
	type args struct {
		a int
		b int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "a is greater than b",
			args: args{a: 5, b: 3},
			want: 5,
		},
		{
			name: "b is greater than a",
			args: args{a: 2, b: 4},
			want: 4,
		},
		{
			name: "a and b are equal",
			args: args{a: 7, b: 7},
			want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := max(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("max() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNzbFile(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "name ends with .nzb",
			args: args{name: "file.nzb"},
			want: true,
		},
		{
			name: "name does not end with .nzb",
			args: args{name: "file.txt"},
			want: false,
		},
		{
			name: "name is empty",
			args: args{name: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNzbFile(tt.args.name); got != tt.want {
				t.Errorf("isNzbFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOriginalNzb(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("original nzb file exists", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		fsInfo := osfs.NewMockFileInfo(ctrl)

		fs.EXPECT().Stat("file-1.nzb").Return(fsInfo, nil).Times(1)
		fs.EXPECT().IsNotExist(nil).Return(false).Times(1)

		if got := getOriginalNzb(fs, "file-1.nzb"); got != "file-1.nzb" {
			t.Errorf("getOriginalNzb() = %v, want %v", got, "file-1.nzb")
		}
	})

	t.Run("original nzb file does not exist", func(t *testing.T) {
		fs := osfs.NewMockFileSystem(ctrl)
		fs.EXPECT().Stat("file-2.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		if got := getOriginalNzb(fs, "file-2.nzb"); got != "" {
			t.Errorf("getOriginalNzb() = %v, want %v", got, "")
		}
	})

}
