package iostep_test

import (
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"testing"

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

	ijs := [][2]int{{0, 15}, {15, 21}, {-1, -1}, {21, 44}}

	for _, ij := range ijs {
		i, j := ij[0], ij[1]

		if i == -1 {
			fmt.Println("Closed.")
			s.Close()
			continue
		}

		output, err := s.Step(input[i:j])
		fmt.Printf("%02d:%02d => %q, %v\n", i, j, output, err)
	}

	// Output:
	// 00:15 => "", <nil>
	// 15:21 => "Hello world!\n", <nil>
	// Closed.
	// 21:44 => "", EOF
}

type dataOnClose struct {
	input   io.Reader
	pending string
}

func (r *dataOnClose) Read(b []byte) (n int, err error) {
	if len(r.pending) == 0 {
		n, err = r.input.Read(b)
		if n > 0 {
			r.pending += string(b[:n])
			err = nil
		}
	}
	if len(r.pending) == 0 && err == io.EOF {
		r.pending += "EOF"
	}
	if len(r.pending) > 0 {
		n = copy(b, r.pending)
		r.pending = r.pending[n:]
	}
	if n == 0 {
		err = io.EOF
	}
	return n, err
}

func TestReaderDataOnClose(t *testing.T) {
	r := iostep.Reader(func(r io.Reader) (io.Reader, error) {
		return &dataOnClose{input: r, pending: "initial..."}, nil
	})


	got, err := r.Step([]byte("stepping..."))
	want := "initial...stepping..."
	if err != nil || string(got) != want {
		t.Fatalf("want %q, got %q with err %v", want, got, err)
	}

	got, err = r.Step([]byte("stepping..."))
	want = "stepping..."
	if err != nil || string(got) != want {
		t.Fatalf("want %q, got %q with err %v", want, got, err)
	}

	r.Close()

	got, err = r.Step(nil)
	want = "EOF"
	if err != nil || string(got) != want {
		t.Fatalf("want %q, got %q with err %v", want, got, err)
	}
}
