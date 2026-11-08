//go:build windows

package main

// The "How to use" card is our own layered window (like the tray menu)
// drawn by ui.HelpCard, instead of the system message box. Esc, the X or
// the OK button close it; the mouse wheel scrolls the body.

import (
	"image"
	"syscall"
	"unsafe"

	"raha/ui"
)

const WM_MOUSEWHEEL = 0x020A

var helpWin struct {
	wnd      uintptr
	card     *ui.HelpCard
	last     *image.RGBA
	x, y     int32
	shown    bool
	tracking bool
	drag     bool
	dragOff  POINT
}

// showHelp opens the card centered on the work area of the monitor the
// mouse is on.
func (a *App) showHelp() {
	if helpWin.wnd == 0 {
		cls := utf16("FalakHelp")
		arrow, _, _ := pLoadCursorW.Call(0, 32512)
		wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(helpProc), HInstance: a.hInst, LpszClassName: cls, HCursor: arrow}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		helpWin.wnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
			uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("Falak help"))), WS_POPUP,
			0, 0, 10, 10, 0, 0, a.hInst, 0)

		logAffinity("help window", helpWin.wnd)
	}
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	var dpiX, dpiY uint32 = 96, 96
	pGetDpiForMonitor.Call(mon, MDT_EFFECTIVE_DPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if dpiX == 0 {
		dpiX = 96
	}
	helpWin.card = ui.NewHelpCard(float64(dpiX)/96, a.fonts, darkMode(), arabic,
		T("How to use", "طريقة الاستخدام"), helpText(), T("OK", "موافق"))
	w, h := helpWin.card.Size()
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	helpWin.x = (mi.RcWork.Left + mi.RcWork.Right - int32(w)) / 2
	helpWin.y = (mi.RcWork.Top + mi.RcWork.Bottom - int32(h)) / 2
	a.refreshHelp()
	pShowWindow.Call(helpWin.wnd, SW_SHOW)
	pSetForegroundWindow.Call(helpWin.wnd)
	helpWin.shown = true
	logf("help card shown at %d,%d (%dx%d)", helpWin.x, helpWin.y, w, h)
}

func hideHelp() {
	if helpWin.shown {
		helpWin.shown = false
		if helpWin.last != nil {
			pDwmFlush.Call()
			animateLayeredClose(helpWin.wnd, helpWin.last, helpWin.x, helpWin.y)
		}
		pShowWindow.Call(helpWin.wnd, SW_HIDE)
	}
}

func (a *App) refreshHelp() {
	img := helpWin.card.Render()
	helpWin.last = img
	presentLayered(helpWin.wnd, img, helpWin.x, helpWin.y)
}

func helpProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic in help window: %v", r)
			hideHelp()
			ret = 0
		}
	}()
	c := helpWin.card
	switch msg {
	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			hideHelp()
		}
		return 0
	case WM_MOUSEWHEEL:
		if c != nil {
			delta := float64(int16((wParam>>16)&0xFFFF)) / 120
			c.Scroll(-delta)
			app.refreshHelp()
		}
		return 0
	case WM_LBUTTONDOWN:
		if c != nil {
			x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
			if h := c.HitTest(x, y); h == 0 { // anywhere but the X, OK and scrollbar
				helpWin.drag = true
				helpWin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
		}
		return 0
	case WM_MOUSEMOVE:
		if helpWin.drag {
			var cur POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
			nx, ny := cur.X-helpWin.dragOff.X, cur.Y-helpWin.dragOff.Y
			if nx != helpWin.x || ny != helpWin.y {
				helpWin.x, helpWin.y = nx, ny
				presentLayered(helpWin.wnd, helpWin.last, helpWin.x, helpWin.y)
			}
			return 0
		}
		if !helpWin.tracking {
			tme := trackMouseEvent{DwFlags: TME_LEAVE, HwndTrack: hwnd}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			helpWin.tracking = true
		}
		return 0
	case WM_MOUSELEAVE:
		helpWin.tracking = false
		return 0
	case 0x0215: // WM_CAPTURECHANGED: the drag has ended
		helpWin.drag = false
		return 0
	case WM_LBUTTONUP:
		if helpWin.drag {
			helpWin.drag = false
			pReleaseCapture.Call()
			return 0
		}
		if c != nil {
			x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
			if h := c.HitTest(x, y); h == 1 || h == 2 {
				hideHelp()
			}
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}
