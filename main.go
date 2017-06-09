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
	if flag.NArg() == 0 {
		args = []string{"cmd", "/c", "start"}
	}
	if spawn {
		os.Exit(start(args))
	}
	os.Exit(server(args))
}
