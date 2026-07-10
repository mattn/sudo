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

func msgWrite(enc *msgEncoder, typ string) io.WriteCloser {
	r, w := io.Pipe()
	go func() {
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
	return w
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
