package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	modshell32         = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteEx = modshell32.NewProc("ShellExecuteExW")
)

const (
	SEE_MASK_NOCLOSEPROCESS = 0x00000040
)

const (
	ERROR_BAD_FORMAT = 11
)

const (
	SE_ERR_FNF             = 2
	SE_ERR_PNF             = 3
	SE_ERR_ACCESSDENIED    = 5
	SE_ERR_OOM             = 8
	SE_ERR_DLLNOTFOUND     = 32
	SE_ERR_SHARE           = 26
	SE_ERR_ASSOCINCOMPLETE = 27
	SE_ERR_DDETIMEOUT      = 28
	SE_ERR_DDEFAIL         = 29
	SE_ERR_DDEBUSY         = 30
	SE_ERR_NOASSOC         = 31
)

type (
	DWORD     uint32
	HANDLE    uintptr
	HINSTANCE HANDLE
	HKEY      HANDLE
	HWND      HANDLE
	ULONG     uint32
	LPCTSTR   uintptr
	LPVOID    uintptr
)

type SHELLEXECUTEINFO struct {
	cbSize         DWORD
	fMask          ULONG
	hwnd           HWND
	lpVerb         LPCTSTR
	lpFile         LPCTSTR
	lpParameters   LPCTSTR
	lpDirectory    LPCTSTR
	nShow          int
	hInstApp       HINSTANCE
	lpIDList       LPVOID
	lpClass        LPCTSTR
	hkeyClass      HKEY
	dwHotKey       DWORD
	hIconOrMonitor HANDLE
	hProcess       HANDLE
}

func ShellExecuteAndWait(hwnd HWND, lpOperation, lpFile, lpParameters, lpDirectory string, nShowCmd int) error {
	var lpctstrVerb, lpctstrParameters, lpctstrDirectory LPCTSTR
	if len(lpOperation) != 0 {
		lpctstrVerb = LPCTSTR(unsafe.Pointer(syscall.StringToUTF16Ptr(lpOperation)))
	}
	if len(lpParameters) != 0 {
		lpctstrParameters = LPCTSTR(unsafe.Pointer(syscall.StringToUTF16Ptr(lpParameters)))
	}
	if len(lpDirectory) != 0 {
		lpctstrDirectory = LPCTSTR(unsafe.Pointer(syscall.StringToUTF16Ptr(lpDirectory)))
	}
	i := &SHELLEXECUTEINFO{
		fMask:        SEE_MASK_NOCLOSEPROCESS,
		hwnd:         hwnd,
		lpVerb:       lpctstrVerb,
		lpFile:       LPCTSTR(unsafe.Pointer(syscall.StringToUTF16Ptr(lpFile))),
		lpParameters: lpctstrParameters,
		lpDirectory:  lpctstrDirectory,
		nShow:        nShowCmd,
	}
	i.cbSize = DWORD(unsafe.Sizeof(*i))
	return ShellExecuteEx(i)
}

func ShellExecuteEx(pExecInfo *SHELLEXECUTEINFO) error {
	ret, _, _ := procShellExecuteEx.Call(uintptr(unsafe.Pointer(pExecInfo)))
	if ret == 1 && pExecInfo.fMask&SEE_MASK_NOCLOSEPROCESS != 0 {
		s, e := syscall.WaitForSingleObject(syscall.Handle(pExecInfo.hProcess), syscall.INFINITE)
		switch s {
		case syscall.WAIT_OBJECT_0:
			break
		case syscall.WAIT_FAILED:
			return os.NewSyscallError("WaitForSingleObject", e)
		default:
			return errors.New("Unexpected result from WaitForSingleObject")
		}
	}
	errorMsg := ""
	if pExecInfo.hInstApp != 0 && pExecInfo.hInstApp <= 32 {
		switch int(pExecInfo.hInstApp) {
		case SE_ERR_FNF:
			errorMsg = "The specified file was not found"
		case SE_ERR_PNF:
			errorMsg = "The specified path was not found"
		case ERROR_BAD_FORMAT:
			errorMsg = "The .exe file is invalid (non-Win32 .exe or error in .exe image)"
		case SE_ERR_ACCESSDENIED:
			errorMsg = "The operating system denied access to the specified file"
		case SE_ERR_ASSOCINCOMPLETE:
			errorMsg = "The file name association is incomplete or invalid"
		case SE_ERR_DDEBUSY:
			errorMsg = "The DDE transaction could not be completed because other DDE transactions were being processed"
		case SE_ERR_DDEFAIL:
			errorMsg = "The DDE transaction failed"
		case SE_ERR_DDETIMEOUT:
			errorMsg = "The DDE transaction could not be completed because the request timed out"
		case SE_ERR_DLLNOTFOUND:
			errorMsg = "The specified DLL was not found"
		case SE_ERR_NOASSOC:
			errorMsg = "There is no application associated with the given file name extension"
		case SE_ERR_OOM:
			errorMsg = "There was not enough memory to complete the operation"
		case SE_ERR_SHARE:
			errorMsg = "A sharing violation occurred"
		default:
			errorMsg = fmt.Sprintf("Unknown error occurred with error code %v", pExecInfo.hInstApp)
		}
	} else {
		return nil
	}
	return errors.New(errorMsg)
}

type msg struct {
	name string
	exit int
	data []byte
}

func msgWrite(enc *gob.Encoder, typ string) io.WriteCloser {
	r, w := io.Pipe()
	go func() {
		defer r.Close()
		var b [256]byte
		for {
			n, err := r.Read(b[:])
			if err != nil {
				break
			}
			enc.Encode(&msg{name: typ, data: b[:n]})
		}
	}()
	return w
}

func client(addr string) int {
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find executable: %v\n", os.Args[0])
		return 1
	}
	defer conn.Close()
	enc, dec := gob.NewEncoder(conn), gob.NewDecoder(conn)

	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)

	// stdin
	inw, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find executable: %v\n", os.Args[0])
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
			switch m.name {
			case "close":
				break in_loop
			case "ctrlc":
				if runtime.GOOS == "windows" {
					// windows doesn't support os.Interrupt
					cmd.Process.Kill()
				} else {
					cmd.Process.Signal(os.Interrupt)
				}
			case "stdin":
				inw.Write(m.data)
			}
		}
	}()

	err = cmd.Run()

	code := 1
	if err != nil {
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			code = status.ExitStatus()
		}
	} else {
		code = 0
	}

	enc.Encode(&msg{name: "exit", exit: code})
	return 0
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

func server() int {
	// make listner to communicate child process
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find executable: %v\n", os.Args[0])
		return 1
	}
	defer lis.Close()

	// make sure executable name to avoid detecting same executable name
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find executable: %v\n", os.Args[0])
		return 1
	}
	args := []string{"-mode", lis.Addr().String()}
	args = append(args, flag.Args()...)

	go func() {
		err = ShellExecuteAndWait(0, "runas", exe, makeCmdLine(args), "", syscall.SW_HIDE)
		if err != nil {
			lis.Close()
		}
	}()

	conn, err := lis.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot execute command: %v\n", makeCmdLine(flag.Args()))
		return 1
	}
	defer conn.Close()

	enc, dec := gob.NewEncoder(conn), gob.NewDecoder(conn)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		for range sc {
			enc.Encode(&msg{name: "ctrlc"})
		}
	}()
	defer close(sc)

	go func() {
		return
		var b [256]byte
		for {
			n, err := os.Stdin.Read(b[:])
			if err != nil {
				enc.Encode(&msg{name: "close"})
				break
			}
			enc.Encode(&msg{name: "stdin", data: b[:n]})
		}
	}()

	for {
		var m msg
		err = dec.Decode(&m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot execute command: %v\n", makeCmdLine(flag.Args()))
			return 1
		}
		switch m.name {
		case "stdout":
			os.Stdout.Write(m.data)
		case "stderr":
			os.Stderr.Write(m.data)
		case "exit":
			return m.exit
		}
	}
	return 0
}

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "mode")
	flag.Parse()
	if mode != "" {
		os.Exit(client(mode))
	}
	os.Exit(server())
}
