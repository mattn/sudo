//go:build windows
// +build windows

package main

import (
	"encoding/gob"
	"io"
	"sync"
	"syscall"
)

type msg struct {
	Name  string
	Exit  int
	Error string
	Data  []byte
}

// msgEncoder is a gob encoder that is safe for concurrent use, since the
// stdout/stderr writers and the main goroutine all encode to a single
// connection.
type msgEncoder struct {
	mu  sync.Mutex
	enc *gob.Encoder
}

func newMsgEncoder(w io.Writer) *msgEncoder {
	return &msgEncoder{enc: gob.NewEncoder(w)}
}

func (e *msgEncoder) Encode(v interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enc.Encode(v)
}

// msgWriter is an io.WriteCloser whose Close blocks until every byte written
// has been encoded, so callers can flush all output before sending a final
// message such as the exit code.
type msgWriter struct {
	w    *io.PipeWriter
	done chan struct{}
	once sync.Once
}

func (mw *msgWriter) Write(p []byte) (int, error) {
	return mw.w.Write(p)
}

func (mw *msgWriter) Close() error {
	err := mw.w.Close()
	mw.once.Do(func() { <-mw.done })
	return err
}

func msgWrite(enc *msgEncoder, typ string) io.WriteCloser {
	r, w := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer r.Close()
		var b [4096]byte
		for {
			n, err := r.Read(b[:])
			if err != nil {
				break
			}
			err = enc.Encode(&msg{Name: typ, Data: b[:n]})
			if err != nil {
				break
			}
		}
	}()
	return &msgWriter{w: w, done: done}
}

func makeCmdLine(args []string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(v)
	}
	return s
}
