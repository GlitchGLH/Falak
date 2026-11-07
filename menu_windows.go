//go:build windows

package main

// The tray menu is our own layered window drawn by ui.Menu (dark or light,
// with on/off switches), instead of the system's plain popup menu.

import (
	"bytes"
	_ "embed"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	xdraw "golang.org/x/image/draw"

	"raha/ui"
)

var (
	pTrackMouseEvent = user32.NewProc("TrackMouseEvent")
)

const (
	WM_ACTIVATE   = 0x0006
	WM_MOUSELEAVE = 0x02A3
	WM_KILLFOCUS  = 0x0008
	SW_SHOW       = 5
	TME_LEAVE     = 0x2
)

type trackMouseEvent struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

var trayMenu struct {
	wnd      uintptr
	m        *ui.Menu
	plain    *ui.Fonts
	x, y     int32 // window position
	anchor   POINT // where the menu was opened
	shown    bool
	tracking bool
}

// lastMenuImg is the menu's last rendered image, kept for the open and
// close fades.
var lastMenuImg *image.RGBA

func (a *App) menuRows() []ui.MenuRow {
	themeSub := T("Light", "فاتح")
	if darkMode() {
		themeSub = T("Dark", "داكن")
	}
	return []ui.MenuRow{
		{ID: "theme", Icon: "moon", Title: T("Dark theme", "الثيم الداكن"), Sub: themeSub, SubBlue: true, Toggle: true, On: darkMode()},
		{ID: "sound", Icon: "sound", Title: T("Wheel sounds", "أصوات العجلة"), Sub: T("Hover and drop sounds", "صوت عند التمرير والإفلات"), Toggle: true, On: soundEnabled},
		{ID: "startup", Icon: "startup", Title: T("Start with Windows", "التشغيل مع بدء ويندوز"), Sub: T("Runs when you sign in", "يعمل عند تسجيل الدخول"), Toggle: true, On: startupEnabled()},
		{ID: "helper", Icon: "shield", Title: T("Key-reading helper", "مساعد قراءة الأزرار"), Sub: T("Administrator", "بصلاحيات المسؤول"), Toggle: true, On: useHelperSetting()},
		{ID: "lang", Icon: "globe", Title: "العربية", Sub: T("Arabic interface", "واجهة عربية"), Toggle: true, On: arabic},
		{ID: "help", Icon: "help", Title: T("How to use", "طريقة الاستخدام"), Footer: true},
		{ID: "quit", Icon: "power", Title: T("Quit", "إنهاء"), Footer: true},
	}
}

func (a *App) createMenuWindow() {
	cls := utf16("FalakMenu")
	arrow, _, _ := pLoadCursorW.Call(0, 32512)
	wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(menuProc), HInstance: a.hInst, LpszClassName: cls, HCursor: arrow}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	trayMenu.wnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("Falak menu"))), WS_POPUP,
		0, 0, 10, 10, 0, 0, a.hInst, 0)
	fontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	trayMenu.plain = ui.LoadFonts([]string{filepath.Join(fontDir, "segoeui.ttf")}, []string{filepath.Join(fontDir, "segoeui.ttf")})
}

// showMenu opens the menu next to the mouse (the tray icon).
func (a *App) showMenu() {
	if trayMenu.wnd == 0 {
		a.createMenuWindow()
	}
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	trayMenu.anchor = pt
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	var dpiX, dpiY uint32 = 96, 96
	pGetDpiForMonitor.Call(mon, MDT_EFFECTIVE_DPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if dpiX == 0 {
		dpiX = 96
	}
	trayMenu.m = &ui.Menu{Scale: float64(dpiX) / 96, Strong: a.fonts, Plain: trayMenu.plain, Hover: -1}
	a.refreshMenu(true)
	if lastMenuImg != nil {
		// Apple-style entrance: fade in while sliding up about 8 px,
		// ~180 ms, ease-out, no bounce. The layout and the final position
		// are exactly refreshMenu's; only the presentation animates.
		slide := int32(math.Round(8 * trayMenu.m.Scale))
		presentLayeredA(trayMenu.wnd, lastMenuImg, trayMenu.x, trayMenu.y+slide, 0)
		pShowWindow.Call(trayMenu.wnd, SW_SHOW)
		for t0 := time.Now(); ; {
			e := time.Since(t0).Seconds() / 0.18
			ee := 1 - math.Pow(1-math.Min(e, 1), 3)
			presentLayeredA(trayMenu.wnd, lastMenuImg, trayMenu.x, trayMenu.y+int32(math.Round(float64(slide)*(1-ee))), uint8(255*ee+0.5))
			if e >= 1 {
				break
			}
			pDwmFlush.Call()
		}
		presentLayered(trayMenu.wnd, lastMenuImg, trayMenu.x, trayMenu.y)
	} else {
		pShowWindow.Call(trayMenu.wnd, SW_SHOW)
	}
	pSetForegroundWindow.Call(trayMenu.wnd)
	trayMenu.shown = true
}

func hideMenu() {
	if trayMenu.shown {
		trayMenu.shown = false
		if lastMenuImg != nil {
			// quick fade out, about 120 ms (clicks are already ignored)
			for t0 := time.Now(); ; {
				e := time.Since(t0).Seconds() / 0.12
				presentLayeredA(trayMenu.wnd, lastMenuImg, trayMenu.x, trayMenu.y, uint8(255*(1-math.Min(e, 1))+0.5))
				if e >= 1 {
					break
				}
				pDwmFlush.Call()
			}
		}
		pShowWindow.Call(trayMenu.wnd, SW_HIDE)
	}
}

// refreshMenu redraws the menu with the current settings (and, when place
// is set, puts it beside the anchor point inside the work area).
func (a *App) refreshMenu(place bool) {
	m := trayMenu.m
	m.Dark, m.RTL = darkMode(), arabic
	m.Logo = menuLogo(int(math.Round(26 * m.Scale)))
	m.Title = T("Falak", "فلك")
	m.Sub = appVersion + " · " + "By UncleGlitch"
	m.Rows = a.menuRows()
	img := m.Render()
	w, h := int32(img.Rect.Dx()), int32(img.Rect.Dy())
	if place {
		pt := trayMenu.anchor
		mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
		mi := MONITORINFO{}
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		off := int32(m.CardOffset())
		cw, ch := w-2*off, h-2*off
		// The card always opens above-left of the mouse — exactly the same
		// position and size in Arabic as in English (only the content is
		// mirrored for RTL), so switching the language never moves it.
		cx, cy := pt.X-cw, pt.Y-ch
		if cy < mi.RcWork.Top { // taskbar at the top: open below the mouse
			cy = pt.Y
		}
		gap := int32(math.Round(6 * m.Scale))
		cx = clamp32(cx, mi.RcWork.Left+gap, mi.RcWork.Right-gap-cw)
		cy = clamp32(cy, mi.RcWork.Top+gap, mi.RcWork.Bottom-gap-ch)
		trayMenu.x, trayMenu.y = cx-off, cy-off
		logf("menu window rect: (%d,%d) %dx%d px, rtl=%v", trayMenu.x, trayMenu.y, w, h, arabic)
	}
	lastMenuImg = img
	presentLayered(trayMenu.wnd, img, trayMenu.x, trayMenu.y)
}

func (a *App) menuAction(id string) {
	switch id {
	case "theme":
		setDarkMode(!darkMode())
	case "sound":
		setSoundOn(!soundEnabled)
		if soundEnabled {
			playHover()
		}
	case "startup":
		setStartup(!startupEnabled())
	case "helper":
		if useHelperSetting() {
			setUseHelper(false) // the helper notices and exits within a second
		} else {
			setUseHelper(true)
			if !helperAlive() {
				hideMenu() // the administrator prompt takes the focus
				a.launchHelper()
				return
			}
		}
	case "lang":
		setLanguage(!arabic)
		a.setTrayTip(idleTip())
		a.refreshMenu(true)
		return
	case "help":
		hideMenu()
		a.showHelp()
		return
	case "quit":
		hideMenu()
		a.quit()
		return
	}
	a.refreshMenu(false)
}

func menuProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic in tray menu: %v", r)
			hideMenu()
			ret = 0
		}
	}()
	m := trayMenu.m
	switch msg {
	case WM_ACTIVATE:
		if wParam&0xFFFF == 0 { // lost the focus: clicked elsewhere
			hideMenu()
		}
		return 0
	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			hideMenu()
		}
		return 0
	case WM_MOUSEMOVE:
		if m == nil {
			return 0
		}
		if !trayMenu.tracking {
			tme := trackMouseEvent{DwFlags: TME_LEAVE, HwndTrack: hwnd}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			trayMenu.tracking = true
		}
		x, y := int16(lParam&0xFFFF), int16((lParam>>16)&0xFFFF)
		if h := m.HitTest(float64(x), float64(y)); h != m.Hover {
			m.Hover = h
			app.refreshMenu(false)
		}
		return 0
	case WM_MOUSELEAVE:
		trayMenu.tracking = false
		if m != nil && m.Hover != -1 {
			m.Hover = -1
			app.refreshMenu(false)
		}
		return 0
	case WM_LBUTTONDOWN:
		return 0
	case WM_LBUTTONUP:
		if m == nil {
			return 0
		}
		x, y := int16(lParam&0xFFFF), int16((lParam>>16)&0xFFFF)
		if i := m.HitTest(float64(x), float64(y)); i >= 0 {
			app.menuAction(m.Rows[i].ID)
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// presentLayered shows img (premultiplied RGBA) on a layered window.
func presentLayered(hwnd uintptr, img *image.RGBA, x, y int32) {
	presentLayeredA(hwnd, img, x, y, 255)
}

// presentLayeredA is presentLayered with a global alpha (0..255) for the
// open and close fades.
func presentLayeredA(hwnd uintptr, img *image.RGBA, x, y int32, alpha uint8) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	screen, _, _ := pGetDC.Call(0)
	defer pReleaseDC.Call(0, screen)
	mem, _, _ := pCreateCompatibleDC.Call(screen)
	defer pDeleteDC.Call(mem)
	bi := BITMAPINFOHEADER{BiWidth: int32(w), BiHeight: -int32(h), BiPlanes: 1, BiBitCount: 32, BiCompression: BI_RGB}
	bi.BiSize = uint32(unsafe.Sizeof(bi))
	var bits uintptr
	dib, _, _ := pCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if dib == 0 {
		return
	}
	defer pDeleteObject.Call(dib)
	old, _, _ := pSelectObject.Call(mem, dib)
	defer pSelectObject.Call(mem, old)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)
	src := img.Pix
	for i := 0; i+3 < len(dst); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
	ptDst := POINT{x, y}
	sz := SIZE{int32(w), int32(h)}
	ptSrc := POINT{}
	blend := BLENDFUNCTION{0, 0, byte(alpha), AC_SRC_ALPHA}
	if r, _, e := pUpdateLayeredWindow.Call(hwnd, screen, uintptr(unsafe.Pointer(&ptDst)), uintptr(unsafe.Pointer(&sz)),
		mem, uintptr(unsafe.Pointer(&ptSrc)), 0, uintptr(unsafe.Pointer(&blend)), ULW_ALPHA); r == 0 {
		logf("menu UpdateLayeredWindow failed: %v", e)
	}
}

// animateLayeredClose plays the same shrink-and-fade for a layered card
// window (the help card, the compress dialog) before it hides.
func animateLayeredClose(wnd uintptr, img *image.RGBA, x, y int32) {
	if img == nil {
		return
	}
	for i := 1; i <= 10; i++ {
		e := float64(i) / 10
		e *= e
		presentLayered(wnd, ui.ZoomFrame(img, 1-0.08*e, 1-e), x, y)
		pDwmFlush.Call()
	}
}

// The app icon for the tray-menu header, from the approved artwork.
//
//go:embed res/icon-1024.png
var iconPNG []byte

var (
	logoSrc  image.Image
	logoOnce sync.Once
	logoByPx = map[int]*image.RGBA{}
)

// menuLogo returns the app icon scaled to px (decoded once, cached).
func menuLogo(px int) *image.RGBA {
	logoOnce.Do(func() {
		im, err := png.Decode(bytes.NewReader(iconPNG))
		if err != nil {
			logf("menu logo: %v", err)
			return
		}
		logoSrc = im
	})
	if logoSrc == nil {
		return nil
	}
	if c, ok := logoByPx[px]; ok {
		return c
	}
	out := image.NewRGBA(image.Rect(0, 0, px, px))
	xdraw.CatmullRom.Scale(out, out.Bounds(), logoSrc, logoSrc.Bounds(), draw.Src, nil)
	logoByPx[px] = out
	return out
}
