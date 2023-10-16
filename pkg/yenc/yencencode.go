package yenc

import (
	"io"
)

// hard code line length
const lineLength = 128

type encoder struct {
	// input
	input []byte
	// output
	output io.Writer
}

func (e *encoder) encode() error {
	var y byte
	count := 0
	lastPos := lineLength - 1

	// make a buffer for the output line
	line := make([]byte, lineLength+3)

	for _, b := range e.input {
		y = byte((b + 42) & 255)

		// NULL, LF, CR, = are critical - TAB/SPACE at the start/end of line are critical - '.' at the start of a line is (sort of) critical
		if y <= 0x3D && ((y == 0x00 || y == 0x0A || y == 0x0D || y == 0x3D) || ((count == 0 || count == lastPos) && (y == 0x09 || y == 0x20)) || (count == 0 && y == 0x2E)) {
			line[count] = '='
			line[count+1] = byte(y + 64)
			count += 2
		} else {
			line[count] = y
			count++
		}

		// end of line?
		if count >= lineLength {
			line[count] = 0x0D
			line[count+1] = 0x0A
			count += 2

			// write the line to the output
			_, err := e.output.Write(line[:count])
			if err != nil {
				return err
			}

			// reset variables
			count = 0
		}
	}

	// dangling count = write CRLF etc
	if count > 0 {
		line[count] = 0x0D
		line[count+1] = 0x0A
		count += 2

		// write the line to the output file
		_, err := e.output.Write(line[:count])
		if err != nil {
			return err
		}
	}

	return nil
}

func Encode(input []byte, output io.Writer) error {
	e := &encoder{input: input, output: output}
	return e.encode()
}
