//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"raha/ui"
)

// Input is detected by polling the key/button state and the cursor position
// about 60 times a second. Earlier builds used global low-level keyboard and
// mouse hooks, but security software (e.g. Kaspersky) silently stops
// delivering input to such hooks in unsigned programs, because that is how
// keyloggers work. Polling only a few keys is ordinary behaviour and is not
// blocked.

const timerPoll = 2

type pollState struct {
	lb, shift, toggle, esc bool
	downPt, lastPt         POINT
	dragging               bool
	blockedLogged          bool
	shiftLogged            int
	pressSeq               int // counts mouse presses, to tell drags apart
}

var ps pollState

func keyDown(vk uintptr) bool {
	r, _, _ := pGetAsyncKeyState.Call(vk)
	return r&0x8000 != 0
}

// leftButtonDown reads the physical state of the (logical) left button.
func leftButtonDown() bool {
	swapped, _, _ := pGetSystemMetrics.Call(23) // SM_SWAPBUTTON
	if swapped != 0 {
		return keyDown(0x02)
	}
	return keyDown(VK_LBUTTON)
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func (a *App) startPolling() {
	pSetTimer.Call(a.msgWnd, timerPoll, 15, 0)
	logf("input polling started")
}

// captureWindow returns the window holding the mouse capture in the
// foreground thread (set during any drag, including file drags), and its class.
func captureWindow() (uintptr, string) {
	gti := GUITHREADINFO{}
	gti.CbSize = uint32(unsafe.Sizeof(gti))
	if r, _, _ := pGetGUIThreadInfo.Call(0, uintptr(unsafe.Pointer(&gti))); r == 0 || gti.HwndCapture == 0 {
		return 0, ""
	}
	buf := make([]uint16, 128)
	n, _, _ := pGetClassNameW.Call(gti.HwndCapture, uintptr(unsafe.Pointer(&buf[0])), 128)
	return gti.HwndCapture, syscall.UTF16ToString(buf[:n])
}

var diagLines int

func diag(format string, args ...any) {
	if diagLines < 400 {
		diagLines++
		logf(format, args...)
	}
}

func (a *App) poll() {
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	h := uint32(0)
	if helperAlive() {
		h = helperMask
	}
	lbAsync := leftButtonDown() || h&keyLB != 0
	capHwnd, capClass := captureWindow()
	held := lbAsync || capHwnd != 0
	shift := keyDown(VK_SHIFT) || h&keyShift != 0
	toggle := keyDown(VK_MENU) || keyDown(VK_CONTROL) || keyDown(VK_TAB) || h&keyTog != 0
	esc := keyDown(VK_ESCAPE) || h&keyEsc != 0

	switch {
	case held && !ps.lb: // pressed
		ps.downPt, ps.dragging = pt, false
		ps.pressSeq++
		diag("press: button=%v capture=%q at %d,%d", lbAsync, capClass, pt.X, pt.Y)
		a.onButtonDown(pt)
	case !held && ps.lb: // released
		diag("release (was dragging: %v)", ps.dragging)
		ps.dragging = false
		a.onButtonUp()
	}
	if held && !ps.dragging {
		cx, _, _ := pGetSystemMetrics.Call(SM_CXDRAG)
		cy, _, _ := pGetSystemMetrics.Call(SM_CYDRAG)
		if abs32(pt.X-ps.downPt.X) > int32(cx) || abs32(pt.Y-ps.downPt.Y) > int32(cy) {
			ps.dragging = true
			diag("drag started (button=%v capture=%q, source %s)", lbAsync, capClass, processOf(a.downHwnd))
			src := a.downHwnd
			if fg, _, _ := pGetForegroundWindow.Call(); src == 0 {
				src = fg
			}
			pPostMessageW.Call(a.msgWnd, wmCheckLevel, src, 0)
			if !lbAsync && capClass == "CLIPBRDWNDCLASS" && !helperAlive() {
				diag("input is hidden from Falak during drags; offering the helper")
				pPostMessageW.Call(a.msgWnd, wmOfferHelp, 0, 0)
			}
		}
	}
	if shift && !ps.shift {
		l, _, _ := pGetAsyncKeyState.Call(VK_LBUTTON)
		r, _, _ := pGetAsyncKeyState.Call(0x02)
		sw, _, _ := pGetSystemMetrics.Call(23)
		diag("Shift pressed: held=%v dragging=%v rawL=%#x rawR=%#x swapped=%d capture=%q wheel=%v",
			held, ps.dragging, uint16(l), uint16(r), sw, capClass, a.wheel != nil)
	}
	// While a conversion shows its progress, only a genuinely new drag (a new
	// mouse press after the drop) may take the wheel; the drag that just
	// ended in the drop must not.
	newDrag := ps.pressSeq != a.dropPress
	busy := a.panel != nil || a.pending != nil
	if held && ps.dragging && shift && a.wheel == nil && (!busy || newDrag) {
		if reason := a.blockedReason(); reason != "" {
			if !ps.blockedLogged {
				ps.blockedLogged = true
				diag("not opening: %s", reason)
			}
		} else {
			logf("Shift+drag detected at %d,%d (source: %s)", pt.X, pt.Y, processOf(a.downHwnd))
			a.panelToBackground()
			a.openWheel(pt)
		}
	}
	if !held {
		ps.blockedLogged = false
	}
	if a.wheel != nil {
		if pt != ps.lastPt {
			a.wheel.SetPointer(float64(pt.X-a.winX), float64(pt.Y-a.winY))
			a.dirty = true
		}
		open := a.wheel.Phase() == ui.PhaseOpen
		// Alt/Ctrl/Tab switches to the tools only when this file has some.
		if toggle && !ps.toggle && open && a.wheel.Revealed() && len(a.paths) > 0 && len(a.itemsFor(1-a.mode)) > 0 {
			a.mode = 1 - a.mode
			a.applyItems(true)
		}
		if esc && !ps.esc && open {
			logf("closed by Esc")
			a.closeWheel()
		}
	}
	ps.lb, ps.shift, ps.toggle, ps.esc, ps.lastPt = held, shift, toggle, esc, pt
}

// blockedReason prevents the wheel from appearing over full-screen games,
// presentations and similar, where Shift + mouse is often used for other things.
func (a *App) blockedReason() string {
	var state int32
	if r, _, _ := pSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state))); r == 0 {
		switch state {
		case 2: // QUNS_BUSY: a full-screen app
			return "a full-screen app is running"
		case 3: // QUNS_RUNNING_D3D_FULL_SCREEN
			return "a full-screen game is running"
		case 4: // QUNS_PRESENTATION_MODE
			return "presentation mode"
		}
	}
	fg, _, _ := pGetForegroundWindow.Call()
	if fg != 0 && processOf(fg) != "explorer.exe" && isFullscreen(fg) {
		return "the active window is full screen"
	}
	return ""
}

func isFullscreen(hwnd uintptr) bool {
	var r RECT
	if ok, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return false
	}
	mon, _, _ := pMonitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)
	mi := MONITORINFO{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	m := mi.RcMonitor
	style, _, _ := pGetWindowLongPtrW.Call(hwnd, ^uintptr(15)) // GWL_STYLE (-16)
	if style&0x00040000 != 0 {                                 // WS_THICKFRAME: a normal resizable (maybe maximized) window
		return false
	}
	return r.Left <= m.Left && r.Top <= m.Top && r.Right >= m.Right && r.Bottom >= m.Bottom
}
