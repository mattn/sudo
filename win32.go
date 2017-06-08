package main

import (
	"errors"
	"fmt"
	"os"
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
type _SHELLEXECUTEINFO struct {
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

// _ShellExecuteAndWait is version of ShellExecuteEx which want process
func _ShellExecuteAndWait(hwnd hwnd, lpOperation, lpFile, lpParameters, lpDirectory string, nShowCmd int) error {
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
	i := &_SHELLEXECUTEINFO{
		fMask:        _SEE_MASK_NOCLOSEPROCESS,
		hwnd:         hwnd,
		lpVerb:       lpctstrVerb,
		lpFile:       lpctstr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpFile))),
		lpParameters: lpctstrParameters,
		lpDirectory:  lpctstrDirectory,
		nShow:        nShowCmd,
	}
	i.cbSize = dword(unsafe.Sizeof(*i))
	return _ShellExecuteEx(i)
}

// ShellExecuteEx is Windows API
func _ShellExecuteEx(pExecInfo *_SHELLEXECUTEINFO) error {
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
