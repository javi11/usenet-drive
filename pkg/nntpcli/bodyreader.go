package nntpcli

import (
	"bytes"
	"io"
)

type bodyReader struct {
	c   *conn
	eof bool
	buf *bytes.Buffer
}

var dotnl = []byte(".\n")
var dotdot = []byte("..")

func (r *bodyReader) Read(p []byte) (n int, err error) {
	if r.eof {
		return 0, io.EOF
	}
	if r.buf == nil {
		r.buf = &bytes.Buffer{}
	}
	if r.buf.Len() == 0 {
		b, err := r.c.r.ReadBytes('\n')
		if err != nil {
			return 0, err
		}
		// canonicalize newlines
		if b[len(b)-2] == '\r' { // crlf->lf
			b = b[0 : len(b)-1]
			b[len(b)-1] = '\n'
		}
		// stop on .
		if bytes.Equal(b, dotnl) {
			r.eof = true
			return 0, io.EOF
		}
		// unescape leading ..
		if bytes.HasPrefix(b, dotdot) {
			b = b[1:]
		}
		r.buf.Write(b)
	}
	n, _ = r.buf.Read(p)
	return
}

func (r *bodyReader) discard() error {
	_, err := io.ReadAll(r)
	return err
}
