package yenc

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSinglepartDecode(t *testing.T) {
	f, err := os.Open("fixtures/singlepart_test.yenc")
	assert.NoError(t, err)

	p, err := Decode(f)
	assert.NoError(t, err)

	expected, err := os.ReadFile("fixtures/singlepart_test_out.jpg")
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(expected, p.Body))
}

func TestMultipartDecode(t *testing.T) {
	f, err := os.Open("fixtures/multipart_test.yenc")
	assert.NoError(t, err)

	p, err := Decode(f)
	assert.NoError(t, err)
	assert.Equal(t, p.Name, "joystick.jpg")

	expected, err := os.ReadFile("fixtures/joystick_out.jpg")
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(expected, p.Body))
}
