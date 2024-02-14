//go:build !windows && !plan9 && !js

package mmap

import (
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/sys/unix"
)

func mmap(f osfs.File, length int) ([]byte, error) {
	return unix.Mmap(int(f.Fd()), 0, length, unix.PROT_READ, unix.MAP_SHARED)
}

func munmap(b []byte) (err error) {
	return unix.Munmap(b)
}
