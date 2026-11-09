//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The details behind "not opening: a full-screen app is running": which
// notification state Windows reports, which window is in front and which
// window covers the monitor. Written to the log once per drag.

const (
	WS_THICKFRAME     = 0x00040000
	WS_EX_TRANSPARENT = 0x00000020
	DWMWA_CLOAKED     = 14
)

// winInfo describes a top-level window for the full-screen check and its log.
type winInfo struct {
	Class   string
	Process string // the exe file name, or "pid N" when it cannot be read
	Rect    RECT
	Style   uint32
	ExStyle uint32
	Own     bool // one of Falak's own windows (this process or the helper)
}

func (w winInfo) String() string {
	var flags []string
	for _, f := range []struct {
		bit  uint32
		name string
	}{{WS_EX_TOPMOST, "topmost"}, {WS_EX_LAYERED, "layered"}, {WS_EX_TRANSPARENT, "transparent"}} {
		if w.ExStyle&f.bit != 0 {
			flags = append(flags, f.name)
		}
	}
	if w.Style&WS_THICKFRAME != 0 {
		flags = append(flags, "thickframe")
	}
	if w.Own {
		flags = append(flags, "falak")
	}
	return fmt.Sprintf("class=%q process=%s rect=%s style=%#x exstyle=%#x [%s]",
		w.Class, w.Process, rectString(w.Rect), w.Style, w.ExStyle, strings.Join(flags, " "))
}

func rectString(r RECT) string {
	return fmt.Sprintf("(%d,%d)-(%d,%d)", r.Left, r.Top, r.Right, r.Bottom)
}

// rectCovers reports whether r covers the whole of m.
func rectCovers(r, m RECT) bool {
	return r.Left <= m.Left && r.Top <= m.Top && r.Right >= m.Right && r.Bottom >= m.Bottom
}

// ownExe is Falak's own exe path; the helper runs from the same file.
var ownExe = func() string {
	p, _ := os.Executable()
	return p
}()

func processPath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func windowInfo(hwnd uintptr) winInfo {
	var w winInfo
	buf := make([]uint16, 256)
	n, _, _ := pGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	w.Class = syscall.UTF16ToString(buf[:n])
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&w.Rect)))
	s, _, _ := pGetWindowLongPtrW.Call(hwnd, ^uintptr(15)) // GWL_STYLE (-16)
	e, _, _ := pGetWindowLongPtrW.Call(hwnd, ^uintptr(19)) // GWL_EXSTYLE (-20)
	w.Style, w.ExStyle = uint32(s), uint32(e)
	var pid uint32
	pGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	path := processPath(pid)
	w.Process = fmt.Sprintf("pid %d", pid)
	if path != "" {
		w.Process = filepath.Base(path)
	}
	w.Own = pid == windows.GetCurrentProcessId() || (path != "" && ownExe != "" && strings.EqualFold(path, ownExe))
	return w
}

// qunsState returns SHQueryUserNotificationState's value; ok is
// false when the call failed.
func qunsState() (state int32, ok bool) {
	r, _, _ := pSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state)))
	return state, r == 0
}

// monitorRectOf returns the rect of the monitor nearest to hwnd, or to the
// cursor when there is no window.
func monitorRectOf(hwnd uintptr) RECT {
	var mon uintptr
	if hwnd != 0 {
		mon, _, _ = pMonitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)
	} else {
		var pt POINT
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		mon, _, _ = pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	}
	mi := MONITORINFO{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	return mi.RcMonitor
}

// enumCover is the state of one firstCoveringWindow search (the poll timer
// runs on one thread, so a single shared value is enough).
var enumCover struct {
	mon   RECT
	found bool
	info  winInfo
}

var enumCoverProc = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
	if v, _, _ := pIsWindowVisible.Call(hwnd); v == 0 {
		return 1
	}
	if ic, _, _ := pIsIconic.Call(hwnd); ic != 0 {
		return 1
	}
	var cloaked uint32
	if r, _, _ := pDwmGetWindowAttribute.Call(hwnd, DWMWA_CLOAKED, uintptr(unsafe.Pointer(&cloaked)), 4); r == 0 && cloaked != 0 {
		return 1
	}
	var rc RECT
	if ok, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc))); ok == 0 || !rectCovers(rc, enumCover.mon) {
		return 1
	}
	w := windowInfo(hwnd)
	if w.Own { // Falak's own windows never count
		return 1
	}
	enumCover.found, enumCover.info = true, w
	return 0 // the first one, top of the Z order, is enough
})

// firstCoveringWindow finds the topmost visible (not minimized, not
// cloaked) window that covers the whole monitor, Falak's own excluded.
func firstCoveringWindow(mon RECT) (winInfo, bool) {
	enumCover.mon, enumCover.found, enumCover.info = mon, false, winInfo{}
	pEnumWindows.Call(enumCoverProc, 0)
	return enumCover.info, enumCover.found
}

// logFullScreenDetails writes what the full-screen check saw.
func logFullScreenDetails() {
	state, ok := qunsState()
	st := "unknown (the call failed)"
	if ok {
		st = fmt.Sprint(state)
	}
	fg, _, _ := pGetForegroundWindow.Call()
	mon := monitorRectOf(fg)
	diag("full-screen check: notification state=%s monitor=%s", st, rectString(mon))
	if fg == 0 {
		diag("full-screen check: no foreground window")
	} else {
		diag("full-screen check: foreground %s", windowInfo(fg))
	}
	if ok && state == 2 { // QUNS_BUSY
		if w, found := firstCoveringWindow(mon); found {
			diag("full-screen check: first visible window covering the monitor: %s", w)
		} else {
			diag("full-screen check: no visible window covers the monitor")
		}
	}
}
