package yenc

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestYencodeText(t *testing.T) {
	// open and read the input file
	inbuf, err := os.ReadFile("fixtures/test1.in")

	assert.NoError(t, err)

	assert.NoError(t, err)

	// generate a dodgy message
	out := new(bytes.Buffer)

	_, err = io.WriteString(out, "=ybegin line=128 size=857 name=test1.in\r\n")
	assert.NoError(t, err)

	err = Encode(inbuf, out)
	assert.NoError(t, err)

	_, err = io.WriteString(out, "=yend size=857 crc32=a3f56400\r\n")
	assert.NoError(t, err)

	// fixture
	expected, err := os.ReadFile("fixtures/test1.yenc")
	assert.NoError(t, err)

	// compare
	assert.True(t, bytes.Equal(expected, out.Bytes()))
}

func TestYencodeBinary(t *testing.T) {
	// open and read the input file
	inbuf, err := os.ReadFile("fixtures/test.in")
	assert.NoError(t, err)

	// open and read the yencode output file
	assert.NoError(t, err)

	// generate a dodgy message
	out := new(bytes.Buffer)

	_, err = io.WriteString(out, "=ybegin line=128 size=153600 name=test.in\r\n")
	assert.NoError(t, err)

	err = Encode(inbuf, out)
	assert.NoError(t, err)

	_, err = io.WriteString(out, "=yend size=153600 crc32=5a9368b3\r\n")
	assert.NoError(t, err)

	// fixture
	expected, err := os.ReadFile("fixtures/test.yenc")
	assert.NoError(t, err)

	// compare
	assert.True(t, bytes.Equal(expected, out.Bytes()))
}

func bench(b *testing.B, n int) {
	inbuf := makeInBuf(n)
	out := new(bytes.Buffer)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i > 0 {
			out.Reset()
		}
		err := Encode(inbuf, out)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(inbuf)))
}

func BenchmarkEncode10(b *testing.B) {
	bench(b, 10)
}

func BenchmarkEncode100(b *testing.B) {
	bench(b, 100)
}

func BenchmarkEncode1000(b *testing.B) {
	bench(b, 1000)
}

func makeInBuf(length int) []byte {
	chars := length * 256 * 132
	pos := 0

	in := make([]byte, chars)
	for i := 0; i < length; i++ {
		for j := 0; j < 256; j++ {
			for k := 0; k < 132; k++ {
				in[pos] = byte(j)
				pos++
			}
		}
	}

	return in
}
