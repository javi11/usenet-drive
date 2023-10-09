package yenc

import (
	"io"
)

// hard code line length to be evil
const lineLength = 128

type encoder struct {
	// input
	input []byte
	// output
	output io.Writer
}

// make a static lookup array
// NOTE: it's actually consistently faster to not use this in my tests at least
//       "AMD Athlon(tm) II X2 240e Processor (2812.59-MHz K8-class CPU)" = ~2% slower
//       "Intel(R) Core(TM) i5-3570K CPU @ 3.40GHz" = ~10% slower
// var yEncTable = makeTable()

// func makeTable() [256]byte {
//     var t [256]byte
//     for i := 0; i < 256; i++ {
//         t[i] = byte((i + 42) & 255)
//     }
//     return t
// }

func (e *encoder) encode() error {
	// misc vars
	var y byte
	count := 0
	lastPos := lineLength - 1

	// make a buffer for the output line
	line := make([]byte, lineLength+3)

	// do yEnc things to the data
	for _, b := range e.input {
		y = byte((b + 42) & 255)
		//y = yEncTable[b]

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
