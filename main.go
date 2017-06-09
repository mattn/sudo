// +build windows

package main

import (
	"flag"
	"os"
)

func main() {
	var mode string
	var spawn bool
	flag.StringVar(&mode, "mode", "", "mode")
	flag.BoolVar(&spawn, "spawn", false, "spawn")
	flag.Parse()

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
