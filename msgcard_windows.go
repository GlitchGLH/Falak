//go:build windows

package main

// Falak's message card replaces the Windows message box (MessageBoxW): the
// same texts, buttons and moment, drawn by ui.MessageCard in Falak's solid
// card and opened and closed with the wheel's animation. Like MessageBox
// it waits until the card is closed and runs its own message loop in the
// meantime, so the windows of the calling thread keep working; it can be
// called from any goroutine (the card lives on that goroutine's thread)
// and from a second Falak process that has no window of its own yet.

import (
	"image"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"raha/ui"
)

const (
	wmClose         = 0x0010
	vkSpace         = 0x20
	vkLeft          = 0x25
	vkRight         = 0x27
	vkC             = 0x43
	cfUnicodeText   = 13
	gmemMoveable    = 0x0002
	lrDefaultSize   = 0x0040
	msgCardExStyle  = WS_EX_LAYERED | WS_EX_TOPMOST // no WS_EX_TOOLWINDOW: on the taskbar, like the message box
	msgCardClass    = "FalakMessage"
	msgCardFallback = 0 // the result of Esc and the X on a card with one button
)

var (
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
)

type msgCardWin struct {
	wnd     uintptr
	card    *ui.MessageCard
	text    string
	last    *image.RGBA
	x, y    int32
	cancel  int // the result of Esc, the X and Alt+F4
	result  int
	done    bool
	drag    bool
	dragOff POINT
}

var (
	msgCardOnce  sync.Once
	msgCardInst  uintptr
	msgCardMu    sync.Mutex
	msgCardByWnd = map[uintptr]*msgCardWin{}
)

// appFontPaths are the bold and semibold faces Falak draws with.
func appFontPaths() (bold, semi []string) {
	dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	return []string{filepath.Join(dir, "segoeuib.ttf"), filepath.Join(dir, "arialbd.ttf")},
		[]string{filepath.Join(dir, "seguisb.ttf"), filepath.Join(dir, "segoeui.ttf")}
}

// messageBox shows text with an OK button and waits until it is closed.
func messageBox(text string, icon uintptr) {
	ic := ui.MsgInfo
	if icon == MB_ICONERROR {
		ic = ui.MsgError
	}
	showMessageCard(text, ic, []string{T("OK", "موافق")}, msgCardFallback)
}

// showMessageCard shows text under Falak's name with the given buttons and
// waits until the user closes it; it returns the index of the button
// pressed, or cancel for Esc, the X and Alt+F4. Enter presses the button
// the keyboard is on (the first one at the start).
func showMessageCard(text string, icon ui.MsgIcon, buttons []string, cancel int) int {
	runtime.LockOSThread() // the window belongs to this thread until it closes
	defer runtime.UnlockOSThread()
	msgCardOnce.Do(func() {
		msgCardInst, _, _ = pGetModuleHandleW.Call(0)
		cls := utf16(msgCardClass)
		arrow, _, _ := pLoadCursorW.Call(0, 32512)
		ico, _, _ := pLoadImageW.Call(msgCardInst, 1, IMAGE_ICON, 0, 0, lrDefaultSize)
		wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(msgCardProc), HInstance: msgCardInst,
			LpszClassName: cls, HCursor: arrow, HIcon: ico}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	var dpiX, dpiY uint32 = 96, 96
	pGetDpiForMonitor.Call(mon, MDT_EFFECTIVE_DPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if dpiX == 0 {
		dpiX = 96
	}
	// a font set of its own: faces are not safe to share across threads
	fonts := ui.LoadFonts(appFontPaths())
	w := &msgCardWin{text: text, cancel: cancel, result: cancel}
	w.card = ui.NewMessageCard(float64(dpiX)/96, fonts, darkMode(), arabic, icon, appName, text, buttons)
	cw, ch := w.card.Size()
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	w.x = (mi.RcWork.Left + mi.RcWork.Right - int32(cw)) / 2
	w.y = (mi.RcWork.Top + mi.RcWork.Bottom - int32(ch)) / 2
	w.wnd, _, _ = pCreateWindowExW.Call(msgCardExStyle, uintptr(unsafe.Pointer(utf16(msgCardClass))),
		uintptr(unsafe.Pointer(utf16(appName))), WS_POPUP, 0, 0, 10, 10, 0, 0, msgCardInst, 0)
	if w.wnd == 0 {
		logf("message card: window not created; message was: %s", text)
		return cancel
	}
	msgCardMu.Lock()
	msgCardByWnd[w.wnd] = w
	msgCardMu.Unlock()
	w.last = w.card.Render()
	logf("message card shown at %d,%d (%dx%d)", w.x, w.y, cw, ch)
	popOpenLayered(w.wnd, w.last, w.x, w.y, func() { pShowWindow.Call(w.wnd, SW_SHOW) })
	pSetForegroundWindow.Call(w.wnd)

	var msg MSG
	for !w.done {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) == 0 { // WM_QUIT: close the card and pass the quit on
			w.close(cancel)
			pPostQuitMessage.Call(msg.WParam)
			break
		}
		if int32(r) < 0 {
			w.close(cancel)
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	logf("message card closed (result %d)", w.result)
	return w.result
}

// close plays the wheel's closing and destroys the card (once).
func (w *msgCardWin) close(result int) {
	if w.done {
		return
	}
	w.done, w.result = true, result
	if w.drag {
		w.drag = false
		pReleaseCapture.Call()
	}
	pDwmFlush.Call()
	animateLayeredClose(w.wnd, w.last, w.x, w.y)
	pShowWindow.Call(w.wnd, SW_HIDE)
	msgCardMu.Lock()
	delete(msgCardByWnd, w.wnd)
	msgCardMu.Unlock()
	pDestroyWindow.Call(w.wnd)
}

func (w *msgCardWin) refresh() {
	w.last = w.card.Render()
	presentLayered(w.wnd, w.last, w.x, w.y)
}

// copyText puts text on the clipboard (Ctrl+C, as in the message box).
func copyText(owner uintptr, text string) {
	u := syscall.StringToUTF16(text)
	if r, _, _ := pOpenClipboard.Call(owner); r == 0 {
		return
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	h, _, _ := pGlobalAlloc.Call(gmemMoveable, uintptr(len(u)*2))
	if h == 0 {
		return
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u)), u)
	pGlobalUnlock.Call(h)
	pSetClipboardData.Call(cfUnicodeText, h)
}

// nextFocus moves the keyboard between the buttons: Tab goes to the next
// one (Shift+Tab back); the arrows move the way they point, so in Arabic,
// where the first button is on the right, Left goes to the next.
func nextFocus(focus, n int, key uintptr, shift, rtl bool) int {
	if n < 2 {
		return focus
	}
	back := false
	switch key {
	case VK_TAB:
		back = shift
	case vkLeft:
		back = !rtl
	case vkRight:
		back = rtl
	default:
		return focus
	}
	if back {
		return (focus + n - 1) % n
	}
	return (focus + 1) % n
}

func msgCardProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	msgCardMu.Lock()
	w := msgCardByWnd[hwnd]
	msgCardMu.Unlock()
	if w == nil || w.done {
		r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	}
	defer func() {
		if r := recover(); r != nil {
			logf("panic in message card: %v", r)
			w.close(w.cancel)
			ret = 0
		}
	}()
	c := w.card
	switch msg {
	case wmClose:
		w.close(w.cancel)
		return 0
	case WM_KEYDOWN:
		switch wParam {
		case VK_ESCAPE:
			w.close(w.cancel)
		case vkReturn, vkSpace:
			w.close(c.Focus)
		case VK_TAB, vkLeft, vkRight:
			shift, _, _ := pGetAsyncKeyState.Call(VK_SHIFT)
			if f := nextFocus(c.Focus, c.Buttons(), wParam, shift&0x8000 != 0, arabic); f != c.Focus {
				c.Focus = f
				w.refresh()
			}
		case vkC:
			if ctrl, _, _ := pGetAsyncKeyState.Call(VK_CONTROL); ctrl&0x8000 != 0 {
				copyText(hwnd, appName+"\r\n\r\n"+w.text)
			}
		}
		return 0
	case WM_MOUSEWHEEL:
		c.Scroll(-float64(int16((wParam>>16)&0xFFFF)) / 120)
		w.refresh()
		return 0
	case WM_LBUTTONDOWN:
		x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
		if c.HitTest(x, y) == ui.MsgHitNone { // drag the card from anywhere else
			w.drag = true
			w.dragOff = POINT{int32(x), int32(y)}
			pSetCapture.Call(hwnd)
		}
		return 0
	case WM_MOUSEMOVE:
		if w.drag {
			var cur POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
			nx, ny := cur.X-w.dragOff.X, cur.Y-w.dragOff.Y
			if nx != w.x || ny != w.y {
				w.x, w.y = nx, ny
				presentLayered(w.wnd, w.last, w.x, w.y)
			}
		}
		return 0
	case 0x0215: // WM_CAPTURECHANGED: the drag has ended
		w.drag = false
		return 0
	case WM_LBUTTONUP:
		if w.drag {
			w.drag = false
			pReleaseCapture.Call()
			return 0
		}
		x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
		switch h := c.HitTest(x, y); {
		case h == ui.MsgHitClose:
			w.close(w.cancel)
		case h >= 0:
			w.close(h)
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}
