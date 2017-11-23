package iostep_test

import (
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"

	"gopkg.in/niemeyer/iostep.v0"
)

func ExampleStepReader() {
	// The compressed input was created by writing "Hello world!\n",
	// flushing the stream, and then writing "Hello once more!\n".
	// The data had 21 bytes after the flush, and has 44 in total.
	zstr := "eJzySM3JyVcozy/KSVHkAgAAAP//gnDy85JTFXLzi1IVuQABAAD//6NgCh8"

	input, err := base64.RawStdEncoding.DecodeString(zstr)
	if err != nil {
		fmt.Printf("bad input: %v", err)
		return
	}

	s := iostep.Reader(func(r io.Reader) (io.Reader, error) {
		return zlib.NewReader(r)
	})

	ijs := [][2]int{{0, 15}, {15, 25}, {25, 30}, {30, 40}, {40, 44}, {0, 0}}

	for _, ij := range ijs {
		i, j := ij[0], ij[1]

		output, err := s.Step(input[i:j])
		fmt.Printf("%02d:%02d => %q, %v\n", i, j, output, err)
	}

	// Output:
	// 00:15 => "", <nil>
	// 15:25 => "Hello world!\n", <nil>
	// 25:30 => "", <nil>
	// 30:40 => "", <nil>
	// 40:44 => "Hello once more!\n", <nil>
	// 00:00 => "", EOF
}

func ExampleStepReader_Close() {
	// The compressed input was created by writing "Hello world!\n",
	// flushing the stream, and then writing "Hello once more!\n".
	// The data had 21 bytes after the flush, and has 44 in total.
	zstr := "eJzySM3JyVcozy/KSVHkAgAAAP//gnDy85JTFXLzi1IVuQABAAD//6NgCh8"

	input, err := base64.RawStdEncoding.DecodeString(zstr)
	if err != nil {
		fmt.Printf("bad input: %v", err)
		return
	}

	s := iostep.Reader(func(r io.Reader) (io.Reader, error) {
		return zlib.NewReader(r)
	})

	ijs := [][2]int{{0, 15}, {15, 25}, {25, 30}, {-1, -1}, {30, 40}, {40, 44}}

	for _, ij := range ijs {
		i, j := ij[0], ij[1]

		if i == -1 {
			s.Close()
			continue
		}

		output, err := s.Step(input[i:j])
		fmt.Printf("%02d:%02d => %q, %v\n", i, j, output, err)
	}

	// Output:
	// 00:15 => "", <nil>
	// 15:25 => "Hello world!\n", <nil>
	// 25:30 => "", <nil>
	// 30:40 => "", EOF
	// 40:44 => "", EOF
}
