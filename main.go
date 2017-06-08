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
	if mode != "" {
		os.Exit(client(mode))
	}
	os.Exit(server())
}
