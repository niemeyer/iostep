package iostep

import (
	"fmt"
	"io"
	"sync"
)


// A StepReader transforms a reader type that processes data in
// a blocking way into a simpler non-blocking interface that
// processes data with a single Step function.
type StepReader struct {
	mu  sync.Mutex
	err error

	newr func(r io.Reader) (io.Reader, error)

	input []byte
	insig *sync.Cond

	output []byte
	outsig *sync.Cond
}

// Reader returns a new stepper that uses the reader returned by
// the provided function to process data. The function will receive
// as a parameter a reader that will make input data available as
// the Step function is called.
//
// If the returned reader also implements io.Closer, its Close
// method will be called when EOF is reached or the stepper's
// Close method is explicitly called.
func Reader(newr func(r io.Reader) (io.Reader, error)) *StepReader {
	s := &StepReader{newr: newr}
	s.insig = sync.NewCond(&s.mu)
	s.outsig = sync.NewCond(&s.mu)
	go s.readLoop()
	return s
}

// Step feeds data through the input reader, and returns all the
// data that was made available by the output reader after that.
//
// The stepper stops waiting for more data from the output reader
// when it is requested for more data from the input reader than
// is available.
func (s *StepReader) Step(data []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil || len(data) == 0 {
		return nil, s.err
	}

	s.output = s.output[:0]
	s.input = data
	s.insig.Signal()
	s.outsig.Wait()

	if len(s.output) > 0 {
		return s.output, nil
	}
	return nil, s.err
}

// Close closes the stepper and also requests the generated output
// reader to be closed asynchronously if it implements io.Closer.
//
// If the Step function is called after the stepper is closed it will
// return the previous error, or io.EOF if there were no errors.
func (s *StepReader) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err == nil {
		s.err = io.EOF
	}
	s.insig.Signal()

	if s.err == io.EOF {
		return nil
	}
	return s.err
}

func (s *StepReader) readLoop() {
	data := make([]byte, 8192)

	r, err := s.newr(&stepReader{s})
	if err != nil {
		s.mu.Lock()
		s.err = err
		s.outsig.Signal()
		s.mu.Unlock()
		return
	}
	for {
		n, err := r.Read(data)
		// This limit should probably be configurable.
		if n+len(s.output) > 1024*1024 {
			n = 0
			err = fmt.Errorf("excessive data on single step")
		}
		s.mu.Lock()
		s.output = append(s.output, data[:n]...)
		if err != nil {
			if s.err == nil {
				s.err = err
			}
			if c, ok := r.(io.Closer); ok {
				err := c.Close()
				if err != nil && s.err == nil {
					s.err = err
				}
			}
		}
		s.mu.Unlock()
		if err != nil {
			break
		}
	}
	s.outsig.Signal()
}

type stepReader struct {
	s *StepReader
}

func (r *stepReader) Read(data []byte) (int, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	for r.s.err == nil && len(r.s.input) == 0 {
		r.s.outsig.Signal()
		r.s.insig.Wait()
	}

	if len(r.s.input) > 0 {
		n := copy(data, r.s.input)
		r.s.input = r.s.input[n:]
		return n, nil
	}

	return 0, r.s.err
}
