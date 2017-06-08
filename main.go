// +build windows

package main

import (
	"flag"
	"os"
)

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "mode")
	flag.Parse()

	args := flag.Args()
	if mode != "" {
		os.Exit(client(mode, args))
	}
	if flag.NArg() == 0 {
		args = []string{"cmd", "/c", "start"}
	}
	os.Exit(server(args))
}
