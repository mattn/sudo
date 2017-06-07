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
	_SEE_MASK_NOCLOSEPROCESS = 0x00000040
)

const (
	_ERROR_BAD_FORMAT = 11
)

const (
	_SE_ERR_FNF             = 2
	_SE_ERR_PNF             = 3
	_SE_ERR_ACCESSDENIED    = 5
	_SE_ERR_OOM             = 8
	_SE_ERR_DLLNOTFOUND     = 32
	_SE_ERR_SHARE           = 26
	_SE_ERR_ASSOCINCOMPLETE = 27
	_SE_ERR_DDETIMEOUT      = 28
	_SE_ERR_DDEFAIL         = 29
	_SE_ERR_DDEBUSY         = 30
	_SE_ERR_NOASSOC         = 31
)

type (
	dword     uint32
	hinstance syscall.Handle
	hkey      syscall.Handle
	hwnd      syscall.Handle
	ulong     uint32
	lpctstr   uintptr
	lpvoid    uintptr
)

// SHELLEXECUTEINFO struct
type SHELLEXECUTEINFO struct {
	cbSize         dword
	fMask          ulong
	hwnd           hwnd
	lpVerb         lpctstr
	lpFile         lpctstr
	lpParameters   lpctstr
	lpDirectory    lpctstr
	nShow          int
	hInstApp       hinstance
	lpIDList       lpvoid
	lpClass        lpctstr
	hkeyClass      hkey
	dwHotKey       dword
	hIconOrMonitor syscall.Handle
	hProcess       syscall.Handle
}

// ShellExecuteAndWait is version of ShellExecuteEx which want process
func ShellExecuteAndWait(hwnd hwnd, lpOperation, lpFile, lpParameters, lpDirectory string, nShowCmd int) error {
	var lpctstrVerb, lpctstrParameters, lpctstrDirectory lpctstr
	if len(lpOperation) != 0 {
		lpctstrVerb = lpctstr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpOperation)))
	}
	if len(lpParameters) != 0 {
		lpctstrParameters = lpctstr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpParameters)))
	}
	if len(lpDirectory) != 0 {
		lpctstrDirectory = lpctstr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpDirectory)))
	}
	i := &SHELLEXECUTEINFO{
		fMask:        _SEE_MASK_NOCLOSEPROCESS,
		hwnd:         hwnd,
		lpVerb:       lpctstrVerb,
		lpFile:       lpctstr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpFile))),
		lpParameters: lpctstrParameters,
		lpDirectory:  lpctstrDirectory,
		nShow:        nShowCmd,
	}
	i.cbSize = dword(unsafe.Sizeof(*i))
	return ShellExecuteEx(i)
}

// ShellExecuteEx is Windows API
func ShellExecuteEx(pExecInfo *SHELLEXECUTEINFO) error {
	ret, _, _ := procShellExecuteEx.Call(uintptr(unsafe.Pointer(pExecInfo)))
	if ret == 1 && pExecInfo.fMask&_SEE_MASK_NOCLOSEPROCESS != 0 {
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
		case _SE_ERR_FNF:
			errorMsg = "The specified file was not found"
		case _SE_ERR_PNF:
			errorMsg = "The specified path was not found"
		case _ERROR_BAD_FORMAT:
			errorMsg = "The .exe file is invalid (non-Win32 .exe or error in .exe image)"
		case _SE_ERR_ACCESSDENIED:
			errorMsg = "The operating system denied access to the specified file"
		case _SE_ERR_ASSOCINCOMPLETE:
			errorMsg = "The file name association is incomplete or invalid"
		case _SE_ERR_DDEBUSY:
			errorMsg = "The DDE transaction could not be completed because other DDE transactions were being processed"
		case _SE_ERR_DDEFAIL:
			errorMsg = "The DDE transaction failed"
		case _SE_ERR_DDETIMEOUT:
			errorMsg = "The DDE transaction could not be completed because the request timed out"
		case _SE_ERR_DLLNOTFOUND:
			errorMsg = "The specified DLL was not found"
		case _SE_ERR_NOASSOC:
			errorMsg = "There is no application associated with the given file name extension"
		case _SE_ERR_OOM:
			errorMsg = "There was not enough memory to complete the operation"
		case _SE_ERR_SHARE:
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
	Name  string
	Exit  int
	Error string
	Data  []byte
}

func msgWrite(enc *gob.Encoder, typ string) io.WriteCloser {
	r, w := io.Pipe()
	go func() {
		defer r.Close()
		var b [4096]byte
		for {
			n, err := r.Read(b[:])
			if err != nil {
				break
			}
			err = enc.Encode(&msg{Name: typ, Data: b[:n]})
			if err != nil {
				break
			}
		}
	}()
	return w
}

func client(addr string) int {
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	enc, dec := gob.NewEncoder(conn), gob.NewDecoder(conn)

	cmd := exec.Command(flag.Arg(0), flag.Args()[1:]...)

	// stdin
	inw, err := cmd.StdinPipe()
	if err != nil {
		enc.Encode(&msg{Name: "error", Error: fmt.Sprintf("cannot execute command: %v", makeCmdLine(flag.Args()))})
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
					cmd.Process.Kill()
				} else {
					cmd.Process.Signal(os.Interrupt)
				}
			case "stdin":
				inw.Write(m.Data)
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

	err = enc.Encode(&msg{Name: "exit", Exit: code})
	if err != nil {
		enc.Encode(&msg{Name: "error", Error: fmt.Sprintf("cannot detect exit code: %v", makeCmdLine(flag.Args()))})
		return 1
	}
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
		err = ShellExecuteAndWait(0, "runas", exe, makeCmdLine(args), "", syscall.SW_HIDE)
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
				enc.Encode(&msg{Name: "close"})
				break
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

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "mode")
	flag.Parse()
	if mode != "" {
		os.Exit(client(mode))
	}
	os.Exit(server())
}
