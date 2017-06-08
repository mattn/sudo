// +build windows

package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func server() int {
	// make listener to communicate child process
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: cannot make listener\n", os.Args[0])
		return 1
	}
	defer lis.Close()

	// make sure executable name to avoid detecting same executable name
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: cannot find executable\n", os.Args[0])
		return 1
	}
	args := []string{"-mode", lis.Addr().String()}
	args = append(args, flag.Args()...)

	var errExec error
	go func() {
		err = _ShellExecuteAndWait(0, "runas", exe, makeCmdLine(args), "", syscall.SW_HIDE)
		if err != nil {
			errExec = err
			lis.Close()
		}
	}()

	conn, err := lis.Accept()
	if err != nil {
		if errExec != nil {
			fmt.Fprintf(os.Stderr, "%v: %v\n", os.Args[0], errExec)
		} else {
			fmt.Fprintf(os.Stderr, "%v: cannot execute command: %v\n", os.Args[0], makeCmdLine(flag.Args()))
		}
		return 1
	}
	defer conn.Close()

	enc, dec := gob.NewEncoder(conn), gob.NewDecoder(conn)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		for range sc {
			enc.Encode(&msg{Name: "ctrlc"})
		}
	}()
	defer close(sc)

	go func() {
		var b [256]byte
		for {
			n, err := os.Stdin.Read(b[:])
			if err != nil {
				// stdin was closed
				if err == io.EOF {
					enc.Encode(&msg{Name: "close"})
				}
				continue
			}
			err = enc.Encode(&msg{Name: "stdin", Data: b[:n]})
			if err != nil {
				break
			}
		}
	}()

	for {
		var m msg
		err = dec.Decode(&m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v: cannot execute command: %v\n", os.Args[0], makeCmdLine(flag.Args()))
			return 1
		}
		switch m.Name {
		case "stdout":
			syscall.Write(syscall.Stdout, m.Data)
		case "stderr":
			syscall.Write(syscall.Stderr, m.Data)
		case "error":
			fmt.Fprintln(os.Stderr, m.Error)
		case "exit":
			return m.Exit
		}
	}
}
