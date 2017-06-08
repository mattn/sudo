// +build windows

package main

import (
	"encoding/gob"
	"io"
	"syscall"
)

type msg struct {
	Name  string
	Exit  int
	Error string
	Data  []byte
}

func msgWrite(enc *gob.Encoder, typ string) io.WriteCloser {
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
