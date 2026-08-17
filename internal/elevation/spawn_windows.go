//go:build windows

package elevation

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ProcessHandle struct {
	handle windows.Handle
}

func (p *ProcessHandle) Wait() error {
	if p == nil || p.handle == 0 {
		return nil
	}
	_, err := windows.WaitForSingleObject(p.handle, windows.INFINITE)
	windows.CloseHandle(p.handle)
	p.handle = 0
	return err
}

func (p *ProcessHandle) Kill() error {
	if p == nil || p.handle == 0 {
		return nil
	}
	err := windows.TerminateProcess(p.handle, 1)
	windows.CloseHandle(p.handle)
	p.handle = 0
	return err
}

func SpawnWorker(socketURI string) (*ProcessHandle, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to locate executable: %w", err)
	}

	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return nil, fmt.Errorf("failed to encode verb: %w", err)
	}
	exePtr, err := windows.UTF16PtrFromString(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to encode exec path: %w", err)
	}
	argsPtr, err := windows.UTF16PtrFromString(fmt.Sprintf("--worker %s", socketURI))
	if err != nil {
		return nil, fmt.Errorf("failed to encode args: %w", err)
	}

	type shellExecuteInfo struct {
		Size          uint32
		Mask          uint32
		Hwnd          windows.Handle
		Verb          *uint16
		File          *uint16
		Parameters    *uint16
		Directory     *uint16
		Show          int32
		InstApp       windows.Handle
		IDList        uintptr
		Class         *uint16
		HkeyClass     windows.Handle
		HotKey        uint32
		IconOrMonitor windows.Handle
		Process       windows.Handle
	}

	const (
		SEE_MASK_NOCLOSEPROCESS = 0x00000040
		SW_HIDE                 = 0
	)

	shell32 := windows.NewLazyDLL("shell32.dll")
	procShellExecuteExW := shell32.NewProc("ShellExecuteExW")

	var info shellExecuteInfo
	info.Size = uint32(unsafe.Sizeof(info))
	info.File = exePtr
	info.Verb = verbPtr
	info.Directory = nil
	info.Show = SW_HIDE
	info.Mask = SEE_MASK_NOCLOSEPROCESS
	info.Parameters = argsPtr

	r1, _, errSys := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return nil, fmt.Errorf("failed to spawn elevated worker via ShellExecuteEx: %w", errSys)
	}

	return &ProcessHandle{handle: info.Process}, nil
}
