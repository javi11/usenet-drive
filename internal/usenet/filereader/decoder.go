package filereader

import "io"

type yencDecoder interface {
	Read(p []byte) (int, error)
	Reset()
	SetReader(reader io.Reader)
	Transform(dst []byte, src []byte, atEOF bool) (nDst int, nSrc int, err error)
}
