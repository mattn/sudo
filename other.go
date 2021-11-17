//go:build !windows
// +build !windows

package main

func main() {
	panic("cannot run on this platform")
}
