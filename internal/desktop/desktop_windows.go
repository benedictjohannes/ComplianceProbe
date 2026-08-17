//go:build windows

package desktop

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	moduser32                     = syscall.NewLazyDLL("user32.dll")
	procGetProcessWindowStation   = moduser32.NewProc("GetProcessWindowStation")
	procGetUserObjectInformationW = moduser32.NewProc("GetUserObjectInformationW")
)

const (
	uoiFlags   = 1
	wsfVisible = 0x0001
)

type userObjectFlags struct {
	fInherit  int32
	fReserved int32
	dwFlags   uint32
}

func isDesktopGUIPlatform() bool {
	hWinStation, _, _ := procGetProcessWindowStation.Call()
	if hWinStation == 0 {
		return false
	}

	var flags userObjectFlags
	var lengthNeeded uint32
	r1, _, _ := procGetUserObjectInformationW.Call(
		hWinStation,
		uintptr(uoiFlags),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Sizeof(flags)),
		uintptr(unsafe.Pointer(&lengthNeeded)),
	)
	if r1 == 0 {
		return false
	}

	return (flags.dwFlags & wsfVisible) != 0
}

func openBrowserPlatform(url string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return cmd.Start()
}
