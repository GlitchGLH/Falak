//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows isolates programs by "integrity level". A normal program cannot
// read the keyboard/mouse state of, or receive drag & drop from, a program
// running with administrator rights. If File Explorer itself was started as
// administrator (this happens when a tweak tool or an elevated Task Manager
// restarts it), Falak sees nothing during drags. The fix is to run Falak at
// the same level.

const (
	ilUnknown = 0
	ilLow     = 0x1000
	ilMedium  = 0x2000
	ilHigh    = 0x3000
)

func ilName(rid uint32) string {
	switch {
	case rid == ilUnknown:
		return "unknown"
	case rid < ilMedium:
		return "low"
	case rid < ilHigh:
		return "medium (normal)"
	case rid < 0x4000:
		return "high (administrator)"
	}
	return "system"
}

func tokenIntegrity(t windows.Token) uint32 {
	var n uint32
	windows.GetTokenInformation(t, windows.TokenIntegrityLevel, nil, 0, &n)
	if n == 0 {
		return ilUnknown
	}
	buf := make([]byte, n)
	if windows.GetTokenInformation(t, windows.TokenIntegrityLevel, &buf[0], n, &n) != nil {
		return ilUnknown
	}
	tml := (*windows.Tokenmandatorylabel)(unsafe.Pointer(&buf[0]))
	sid := tml.Label.Sid
	c := sid.SubAuthorityCount()
	if c == 0 {
		return ilUnknown
	}
	return sid.SubAuthority(uint32(c) - 1)
}

func selfIntegrity() uint32 {
	return tokenIntegrity(windows.GetCurrentProcessToken())
}

// processIntegrity returns the integrity of the process owning hwnd.
// denied=true means Windows refused to tell us, which only happens when that
// process runs at a higher level than ours.
func processIntegrity(hwnd uintptr) (rid uint32, denied bool) {
	root, _, _ := pGetAncestor.Call(hwnd, GA_ROOT)
	var pid uint32
	pGetWindowThreadProcessId.Call(root, uintptr(unsafe.Pointer(&pid)))
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ilUnknown, true
	}
	defer windows.CloseHandle(h)
	var t windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &t); err != nil {
		return ilUnknown, true
	}
	defer t.Close()
	return tokenIntegrity(t), false
}

var (
	mutexHandle    uintptr
	checkedSources = map[string]bool{}
)

// checkSourceLevel is called when a drag starts. If the drag source runs at a
// higher level than Falak, it explains the problem once and offers to restart
// Falak as administrator.
func (a *App) checkSourceLevel(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	name := processOf(hwnd)
	rid, denied := processIntegrity(hwnd)
	mine := selfIntegrity()
	key := fmt.Sprintf("%s/%d/%v", name, rid, denied)
	if !checkedSources[key] {
		checkedSources[key] = true
		logf("drag source %s: level=%s denied=%v; Falak level=%s", name, ilName(rid), denied, ilName(mine))
	}
}
