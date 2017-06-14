package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func start(args []string) int {
	if exe, err := exec.LookPath(args[0]); err == nil {
		args[0] = exe
	}
	if err := _ShellExecuteNowait(0, "runas", args[0], makeCmdLine(args[1:]), "", syscall.SW_NORMAL); err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", os.Args[0], err)
		return 1
	}
	return 0
}
