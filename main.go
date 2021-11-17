//go:build windows
// +build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

const name = "sudo"

const version = "0.0.2"

var revision = "HEAD"

func main() {
	var mode string
	var spawn bool
	var showVersion bool
	flag.StringVar(&mode, "mode", "", "mode")
	flag.BoolVar(&spawn, "spawn", false, "spawn")
	flag.BoolVar(&showVersion, "V", false, "print the version")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}
	args := flag.Args()
	if mode != "" {
		os.Exit(client(mode, args))
	}
	if spawn {
		if flag.NArg() == 0 {
			args = []string{"cmd"}
		}
		os.Exit(start(args))
	}
	if flag.NArg() == 0 {
		args = []string{"cmd", "/c", "start"}
	}
	os.Exit(server(args))
}
