// +build windows

package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

func client(addr string, args []string) int {
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	enc, dec := gob.NewEncoder(conn), gob.NewDecoder(conn)

	cmd := exec.Command(args[0], args[1:]...)

	// stdin
	inw, err := cmd.StdinPipe()
	if err != nil {
		enc.Encode(&msg{Name: "error", Error: fmt.Sprintf("cannot execute command: %v", makeCmdLine(args))})
		return 1
	}
	defer inw.Close()

	// stdout
	outw := msgWrite(enc, "stdout")
	defer outw.Close()
	cmd.Stdout = outw

	// stderr
	errw := msgWrite(enc, "stderr")
	defer errw.Close()
	cmd.Stderr = errw

	go func() {
		defer inw.Close()
	in_loop:
		for {
			var m msg
			err = dec.Decode(&m)
			if err != nil {
				return
			}
			switch m.Name {
			case "close":
				break in_loop
			case "ctrlc":
				if runtime.GOOS == "windows" {
					// windows doesn't support os.Interrupt
					exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
				} else {
					cmd.Process.Signal(os.Interrupt)
				}
				break in_loop
			case "stdin":
				inw.Write(m.Data)
			}
		}
	}()

	var environ []string
	err = dec.Decode(&environ)
	if err != nil {
		enc.Encode(&msg{Name: "error", Error: fmt.Sprintf("cannot execute command: %v", makeCmdLine(args))})
		return 1
	}
	cmd.Env = environ

	err = cmd.Run()

	code := 1
	if err != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			code = status.ExitStatus()
		}
	} else {
		code = 0
	}

	err = enc.Encode(&msg{Name: "exit", Exit: code})
	if err != nil {
		enc.Encode(&msg{Name: "error", Error: fmt.Sprintf("cannot detect exit code: %v", makeCmdLine(args))})
		return 1
	}
	return 0
}
