//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"raha/conv"
	"raha/ui"
)

var appName, appVersion string

const (
	appVersionNum = "1.0"
	wmTray        = WM_APP + 1
	timerAnim     = 1
	menuHelp      = 2
	menuStartup   = 3
	menuQuit      = 4
	menuLog       = 5
	menuAdmin     = 6
	menuSound     = 7
	menuArabic    = 8
	wmCheckLevel  = WM_APP + 3
	wmOfferHelp   = WM_APP + 4
	wmJobDone     = WM_APP + 6
	wmJobProgress = WM_APP + 7
	wmFrame       = WM_APP + 8
	wmQuitUpdate  = WM_APP + 9 // a newer copy is starting: close so it can replace us
	runKeyPath    = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName  = "Falak"
)

type result struct {
	job     int
	ok      bool
	errText string
	count   int
	label   string
}

type thumbMsg struct {
	path  string
	thumb *image.RGBA
}

type App struct {
	hInst   uintptr
	overlay uintptr
	msgWnd  uintptr
	fonts   *ui.Fonts

	downHwnd uintptr

	wheel         *ui.Wheel
	winX, winY    int32
	paths         []string
	mode          int
	dropped       bool
	closeDeadline time.Time
	finishedAt    time.Time
	openedAt      time.Time
	dirty         bool
	timerOn       bool
	pumpStop      chan struct{}
	framePending  atomic.Bool
	lastCur       POINT
	lastHover     int
	dropPress     int // the mouse press whose drag ended in the last drop

	memDC, dib, bits uintptr
	dibW, dibH       int

	results chan result
	thumbs  chan thumbMsg

	jobSeq     int
	background int // jobs running without their progress window

	panel     *ui.Panel // the small "Converting to …" window
	panelJob  int       // job shown by the panel (0 = none)
	pending   *panelReq // panel to show once the wheel has vanished
	cancels   map[int]context.CancelFunc
	cancelled map[int]bool
	lastPanel time.Time
	dragPanel bool  // the progress window is being moved
	dragOff   POINT // cursor offset inside it

	lastPresented *image.RGBA // the last frame shown, for the close animation
	closingPanel  bool        // the close animation is running
}

type panelReq struct {
	job         int
	title, file string
	cx, cy      int32 // screen point the panel is centred on
	scale       float64
}

// jobProgress holds each running job's progress: 0..1, or -1 while unknown.
var jobProgress sync.Map // job number -> float64

// engineMu lets only one job at a time download an engine (FFmpeg, PDFium …),
// so two conversions started together never write the same files.
var engineMu sync.Mutex

var taskbarCreated uintptr // message sent when the taskbar (Explorer) restarts

var app = &App{results: make(chan result, 32), thumbs: make(chan thumbMsg, 8),
	cancels: map[int]context.CancelFunc{}, cancelled: map[int]bool{}}

func main() {
	runtime.LockOSThread()
	openLog()
	loadLanguage()
	if len(os.Args) > 1 && os.Args[1] == "--keys" {
		arg := ""
		if len(os.Args) > 2 {
			arg = os.Args[2]
		}
		runKeysHelper(arg)
		return
	}
	logf("Falak %s starting, args=%v", appVersion, os.Args[1:])
	if sw, _, _ := pGetSystemMetrics.Call(23); sw != 0 {
		logf("note: primary/secondary mouse buttons are swapped in Windows settings")
	}
	logf("Falak runs at level: %s", ilName(selfIntegrity()))
	pSetProcessDpiAwarenessContext.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)

	mutexName := utf16(`Local\FalakFileWheel`)
	var mutexErr error
	mutexHandle, _, mutexErr = pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	if mutexErr == windows.ERROR_ALREADY_EXISTS {
		if !runningFromInstall() && replaceRunningCopy() {
			logf("the running copy closed; this one takes over")
		} else {
			logf("another instance is running; exiting")
			messageBox(T("Falak is already running in the background. Its icon is next to the clock on the taskbar.", "فلك يعمل بالفعل في الخلفية. تجد أيقونته بجانب الساعة في شريط المهام."), MB_ICONINFO)
			return
		}
	}
	removeOldPrototype()
	if selfIntegrity() >= ilHigh && relaunchNormal() {
		return
	}
	if ensureInstalled() {
		return
	}
	refreshStartup()
	cleanOldCopies()

	pOleInitialize.Call(0)
	app.hInst, _, _ = pGetModuleHandleW.Call(0)
	fontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	app.fonts = ui.LoadFonts(
		[]string{filepath.Join(fontDir, "segoeuib.ttf"), filepath.Join(fontDir, "arialbd.ttf")},
		[]string{filepath.Join(fontDir, "seguisb.ttf"), filepath.Join(fontDir, "segoeui.ttf")},
	)

	warmUp(app.fonts) // synchronous: font faces are not safe to share across goroutines
	soundEnabled = soundOn()
	conv.HidePath = hidePath
	conv.Logf = logf
	conv.RarHint = winrarFromRegistry
	// Engines live in Falak's own data folder (removed whole by the
	// uninstaller), never in the install folder, which the user may have
	// chosen and may share with other files. Earlier builds used
	// <default install folder>\tools; those copies are removed (the pinned
	// engines are downloaded again anyway).
	conv.ToolsDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Falak", "tools")
	go os.RemoveAll(filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Falak", "tools"))
	conv.PrepareCmd = func(c *exec.Cmd) {
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	}
	app.createWindows()
	if hr, _, _ := pRegisterDragDrop.Call(app.overlay, uintptr(unsafe.Pointer(theDropTarget))); hr != 0 {
		logf("RegisterDragDrop failed: %#x", hr)
		messageBox(fmt.Sprintf(T("Couldn't enable drag and drop (0x%X).", "تعذّر تفعيل السحب والإفلات (0x%X)."), hr), MB_ICONERROR)
		return
	}
	taskbarCreated, _, _ = pRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16("TaskbarCreated"))))
	app.addTray()
	app.startPolling()
	if useHelperSetting() {
		app.launchHelper()
	}

	startedAtLogin := len(os.Args) > 1 && os.Args[1] == "--startup"
	if firstRun() {
		app.balloon(T("Falak is running", "فلك يعمل الآن"), T("Drag a file and press Shift while dragging to bring up the wheel.", "اسحب ملفًا واضغط Shift أثناء السحب لتظهر العجلة."), false)
	} else if !startedAtLogin {
		app.balloon(T("Falak is running in the background", "فلك يعمل في الخلفية"), T("Drag a file and press Shift while dragging.", "اسحب ملفًا واضغط Shift أثناء السحب."), false)
	}

	// FALAK_DEMO=<image path> opens the wheel at the screen center for testing
	// without a real drag.
	if demo := os.Getenv("FALAK_DEMO"); demo != "" {
		pt := POINT{640, 400}
		app.openWheel(pt)
		app.onDragEnter([]string{demo}, POINT{pt.X + 70, pt.Y - 120})
		if os.Getenv("FALAK_DEMO_TOOLS") != "" {
			app.mode = 1
			app.applyItems(true)
		}
		if os.Getenv("FALAK_DEMO_DROP") != "" {
			app.onDrop(nil, POINT{pt.X + 70, pt.Y - 120})
			if ms, _ := strconv.Atoi(os.Getenv("FALAK_DEMO_CANCEL")); ms > 0 {
				go func() { // press the X after ms milliseconds (testing)
					time.Sleep(time.Duration(ms) * time.Millisecond)
					s := 1.0
					if app.panel != nil {
						s = app.panel.Scale
					}
					x, y := int((ui.PanelMargin+32.5)*s), int((ui.PanelMargin+32.5)*s)
					pPostMessageW.Call(app.overlay, WM_LBUTTONDOWN, 1, uintptr(y<<16|x))
				}()
			}
		}
	}

	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	logf("exiting")
	app.removeTray()
}

func messageBox(text string, icon uintptr) {
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16(text))), uintptr(unsafe.Pointer(utf16(appName))),
		MB_OK|icon|rtl()|MB_TOPMOST)
}

// removeOldPrototype drops the startup entry of the earlier "Raha" build.
func removeOldPrototype() {
	if k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE); err == nil {
		k.DeleteValue("Raha")
		k.Close()
	}
}

// ---------- windows ----------

func (a *App) createWindows() {
	overlayClass := utf16("FalakWheel")
	arrow, _, _ := pLoadCursorW.Call(0, 32512) // IDC_ARROW
	wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(overlayProc), HInstance: a.hInst, LpszClassName: overlayClass, HCursor: arrow}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	a.overlay, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE,
		uintptr(unsafe.Pointer(overlayClass)), uintptr(unsafe.Pointer(utf16("Falak"))), WS_POPUP,
		0, 0, 10, 10, 0, 0, a.hInst, 0)

	// The glass is a blurred screen capture. The windows are layered
	// (UpdateLayeredWindow), and captureScreen BitBlts the screen DC with
	// plain SRCCOPY and no CAPTUREBLT, which excludes layered windows, so
	// the glass never shows the window itself. No WDA_EXCLUDEFROMCAPTURE:
	// it makes other apps (NVIDIA App) refuse desktop capture.

	msgClass := utf16("FalakMsg")
	mc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(msgProc), HInstance: a.hInst, LpszClassName: msgClass}
	mc.CbSize = uint32(unsafe.Sizeof(mc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&mc)))
	a.msgWnd, _, _ = pCreateWindowExW.Call(WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(msgClass)),
		uintptr(unsafe.Pointer(utf16("FalakMsg"))), WS_POPUP, 0, 0, 0, 0, 0, 0, a.hInst, 0)
	logf("windows created: overlay=%#x msg=%#x", a.overlay, a.msgWnd)
	logAffinity("wheel overlay", a.overlay)
	logAffinity("msg window", a.msgWnd)
}

// logAffinity logs the display affinity of a Falak window: it must always
// be 0 (WDA_NONE). Falak never hides its windows from screen capture.
func logAffinity(name string, hwnd uintptr) {
	var aff uint32
	r, _, _ := pGetWindowDisplayAffinity.Call(hwnd, uintptr(unsafe.Pointer(&aff)))
	if r == 0 {
		logf("%s: GetWindowDisplayAffinity failed", name)
		return
	}
	if aff != 0 {
		logf("%s: display affinity %d - SHOULD BE 0 (WDA_NONE)", name, aff)
		return
	}
	logf("%s: display affinity 0 (WDA_NONE)", name)
}

func overlayProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic handling message %#x: %v", msg, r)
			// Never leave a broken wheel on screen; reset so it can open again.
			pShowWindow.Call(app.overlay, SW_HIDE)
			app.wheel = nil
			// the job keeps running; it finishes with a notification
			if app.panel != nil || app.pending != nil {
				app.background++
			}
			app.panel, app.panelJob, app.pending, app.dragPanel = nil, 0, nil, false
			app.syncFlags()
			ret = 0
		}
	}()
	switch msg {
	case WM_TIMER, wmFrame:
		if msg == wmFrame {
			app.framePending.Store(false)
		}
		app.tick()
		return 0
	case WM_MOUSEACTIVATE:
		return MA_NOACTIVATE
	case WM_LBUTTONDOWN:
		if app.panel != nil {
			x, y := int16(lParam&0xFFFF), int16((lParam>>16)&0xFFFF)
			if app.panel.OverClose(float64(x), float64(y)) {
				app.cancelPanel()
			} else {
				// anywhere else: move the window with the mouse
				app.dragPanel = true
				app.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
		}
		return 0
	case WM_MOUSEMOVE:
		if app.dragPanel && app.panel != nil {
			app.movePanel()
		}
		return 0
	case WM_LBUTTONUP:
		if app.dragPanel {
			pReleaseCapture.Call()
		}
		return 0
	case 0x0215: // WM_CAPTURECHANGED: the move has ended
		if app.dragPanel {
			app.dragPanel = false
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

func msgProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic in poll/tray: %v", r)
			ret = 0
		}
	}()
	if msg == WM_TIMER && wParam == timerPoll {
		app.poll()
		return 0
	}
	if msg == wmJobDone {
		app.handleResults()
		return 0
	}
	if msg == wmQuitUpdate {
		logf("a newer copy is starting; closing")
		app.quit()
		return 0
	}
	if taskbarCreated != 0 && msg == taskbarCreated {
		logf("taskbar restarted; adding the tray icon again")
		app.addTray()
		return 0
	}
	if msg == wmJobProgress {
		if int(wParam) != app.panelJob || app.panel == nil {
			app.setTrayTip(appName + " — " + jobProgressText(int(wParam)))
		}
		return 0
	}
	if msg == wmKeys {
		helperMask = uint32(wParam)
		if !helperAlive() {
			logf("helper connected")
		}
		helperSeen = time.Now()
		return 0
	}
	if msg == wmOfferHelp {
		app.offerHelper()
		return 0
	}
	if msg == wmCheckLevel {
		app.checkSourceLevel(wParam)
		return 0
	}
	if msg == wmTray {
		switch loword(lParam) {
		case WM_RBUTTONUP, WM_LBUTTONUP:
			app.showMenu()
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// syncFlags is kept for clarity at state changes (input is polled on this thread).
func (a *App) syncFlags() {}

// ---------- input events (from the poller) ----------

func (a *App) onButtonDown(pt POINT) {
	a.downHwnd, _, _ = pWindowFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))))
	if a.wheel != nil && a.wheel.Phase() == ui.PhaseOpen && !a.dropped {
		a.closeWheel()
	}
}

func (a *App) onButtonUp() {
	if a.wheel != nil && a.wheel.Phase() == ui.PhaseOpen && !a.dropped && a.closeDeadline.IsZero() {
		// Give OLE a moment to deliver Drop before closing.
		a.closeDeadline = time.Now().Add(300 * time.Millisecond)
	}
}

func processOf(hwnd uintptr) string {
	if hwnd == 0 {
		return "?"
	}
	root, _, _ := pGetAncestor.Call(hwnd, GA_ROOT)
	var pid uint32
	pGetWindowThreadProcessId.Call(root, uintptr(unsafe.Pointer(&pid)))
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return fmt.Sprintf("pid %d", pid)
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return fmt.Sprintf("pid %d", pid)
	}
	return filepath.Base(windows.UTF16ToString(buf[:n]))
}

// ---------- wheel lifecycle ----------

// panelToBackground lets the running job finish without its window, so a
// new drag can use the screen. The tray icon shows the progress and a
// notification follows.
func (a *App) panelToBackground() {
	job := a.panelJob
	if a.pending != nil {
		job = a.pending.job
		a.pending = nil
	}
	if job == 0 {
		return
	}
	logf("job %d continues in the background (new drag)", job)
	a.background++
	a.hidePanel()
	a.balloon(T("Still converting in the background", "يستمر التحويل في الخلفية"), T("You'll get a notification when it's done. Follow the progress on the Falak icon next to the clock.", "سيظهر إشعار عند الانتهاء. تابع التقدّم من أيقونة فلك بجانب الساعة."), false)
}

// hidePanel removes the progress window at once (no animation).
func (a *App) hidePanel() {
	if a.panel != nil {
		pShowWindow.Call(a.overlay, SW_HIDE)
	}
	if a.dragPanel {
		a.dragPanel = false
		pReleaseCapture.Call()
	}
	a.panel, a.panelJob = nil, 0
	if a.wheel == nil {
		a.stopTimer()
	}
}

// cancelPanel stops the job shown in the panel (the X button).
func (a *App) cancelPanel() {
	job := a.panelJob
	if job == 0 {
		return
	}
	logf("job %d cancelled by the user", job)
	a.cancelled[job] = true
	if c := a.cancels[job]; c != nil {
		c()
	}
	a.closingPanel = true
	a.animatePanelClose()
	a.closingPanel = false
	a.hidePanel()
	a.setTrayTip(idleTip())
}

// showPanel pops up the progress window where the wheel was.
func (a *App) showPanel() {
	req := a.pending
	a.pending = nil
	if req == nil || a.cancels[req.job] == nil {
		return // already finished while the wheel vanished
	}
	pt := POINT{req.cx, req.cy}
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	mi := MONITORINFO{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	pw, ph := ui.PanelSize(req.scale)
	x := clamp32(req.cx-int32(pw)/2, mi.RcWork.Left, mi.RcWork.Right-int32(pw))
	y := clamp32(req.cy-int32(ph)/2, mi.RcWork.Top, mi.RcWork.Bottom-int32(ph))
	now := time.Now()
	a.panel = ui.NewPanelTheme(req.scale, nil, a.fonts, req.title, req.file, darkMode(), now)
	a.panelJob = req.job
	a.panel.SetProgress(jobProgressOf(req.job))
	a.winX, a.winY = x, y
	a.setClickThrough(false)
	a.present(a.panel.Render(now))
	a.lastPanel = now
	pSetWindowPos.Call(a.overlay, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOACTIVATE|SWP_NOSIZE|SWP_NOMOVE)
	pShowWindow.Call(a.overlay, SW_SHOWNOACTIVATE)
	if !a.timerOn {
		pTimeBeginPeriod.Call(1)
		a.startPump()
		a.timerOn = true
	}
	logf("progress window shown at %d,%d (%dx%d) in %v", x, y, pw, ph, time.Since(now).Round(time.Millisecond))
}

// movePanel follows the cursor while the window is dragged.
func (a *App) movePanel() {
	var cur POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
	x, y := cur.X-a.dragOff.X, cur.Y-a.dragOff.Y
	if x == a.winX && y == a.winY {
		return
	}
	a.winX, a.winY = x, y
	now := time.Now()
	a.present(a.panel.Render(now))
	a.lastPanel = now
}


func (a *App) tickPanel() {
	if a.closingPanel {
		return
	}
	now := time.Now()
	var cur POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
	a.panel.SetPointer(float64(cur.X-a.winX), float64(cur.Y-a.winY))
	a.panel.SetProgress(jobProgressOf(a.panelJob))
	// full rate while popping in, then ~60 fps is plenty for a bar
	if !a.panel.Animating(now) && now.Sub(a.lastPanel) < 15*time.Millisecond {
		return
	}
	a.lastPanel = now
	a.present(a.panel.Render(now))
}

// setClickThrough makes the wheel window ignore the mouse (while it only
// shows progress) or take part in drag and drop again.
func (a *App) setClickThrough(on bool) {
	const gwlExStyle = ^uintptr(19) // -20
	const wsExTransparent = 0x20
	st, _, _ := pGetWindowLongPtrW.Call(a.overlay, gwlExStyle)
	if on {
		st |= wsExTransparent
	} else {
		st &^= wsExTransparent
	}
	pSetWindowLongPtrW.Call(a.overlay, gwlExStyle, st)
}

func (a *App) openWheel(pt POINT) {
	a.setClickThrough(false)
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	mi := MONITORINFO{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	var dpiX, dpiY uint32 = 96, 96
	pGetDpiForMonitor.Call(mon, MDT_EFFECTIVE_DPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if dpiX == 0 {
		dpiX = 96
	}
	scale := float64(dpiX) / 96
	size := int32(math.Round(ui.LogicalSize * scale))

	// The window is kept inside the work area (the screen without the
	// taskbar) and the wheel is always drawn in its middle, so near an edge
	// the whole wheel moves inward instead of being cut off by the edge.
	area := mi.RcWork
	if area.Right-area.Left < size || area.Bottom-area.Top < size {
		area = mi.RcMonitor
	}
	x := clamp32(pt.X-size/2, area.Left, area.Right-size)
	y := clamp32(pt.Y-size/2, area.Top, area.Bottom-size)
	a.winX, a.winY = x, y

	t0 := time.Now()
	shot := captureScreen(x, y, size, size)
	t1 := time.Now()
	frost := ui.FrostBackground(shot, 14*scale, 1.25)
	t2 := time.Now()
	now := time.Now()
	a.wheel = ui.New(scale, int(size), int(size), float64(size)/2, float64(size)/2, frost, darkMode(), a.fonts, now)
	a.wheel.Sharp = ui.FrostBackground(shot, 3*scale, 1.1)
	a.wheel.Prepare()              // blurred copies, in the background
	a.wheel.PrebuildGlass(7, 8, 9) // the usual segment counts, ready ahead
	a.wheel.SetPointer(float64(pt.X-x), float64(pt.Y-y))
	a.paths, a.mode, a.dropped = nil, 0, false
	a.lastHover = -1
	a.closeDeadline, a.finishedAt, a.openedAt = time.Time{}, time.Time{}, now
	a.drainThumbs()
	a.syncFlags()

	a.present(a.wheel.Render(now))
	t3 := time.Now()
	pSetWindowPos.Call(a.overlay, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOACTIVATE|SWP_NOSIZE|SWP_NOMOVE)
	pShowWindow.Call(a.overlay, SW_SHOWNOACTIVATE)
	if !a.timerOn {
		pTimeBeginPeriod.Call(1)
		a.startPump()
		a.timerOn = true
	}
	nudgeMouse()
	logf("wheel window shown at %d,%d size %d (scale %.2f); capture %v, frost %v, render %v, total %v",
		x, y, size, scale, t1.Sub(t0).Round(time.Millisecond), t2.Sub(t1).Round(time.Millisecond), t3.Sub(t2).Round(time.Millisecond), time.Since(t0).Round(time.Millisecond))
}

// nudgeMouse injects a zero-distance mouse move so the drag loop notices our
// window right away instead of waiting for the user to move the mouse.
func nudgeMouse() {
	in := INPUT{Type: 0}
	in.Mi.DwFlags = 0x0001 // MOUSEEVENTF_MOVE
	pSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func clamp32(v, lo, hi int32) int32 {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

func (a *App) stopTimer() {
	if a.timerOn {
		close(a.pumpStop)
		a.pumpStop = nil
		pTimeEndPeriod.Call(1)
		a.timerOn = false
	}
}

func (a *App) closeWheel() {
	if a.wheel != nil {
		a.wheel.Close(time.Now())
		a.syncFlags()
	}
}

func (a *App) drainThumbs() {
	for {
		select {
		case <-a.thumbs:
		default:
			return
		}
	}
}

func (a *App) tick() {
	if a.wheel == nil {
		if a.panel != nil {
			a.tickPanel()
			return
		}
		a.stopTimer()
		return
	}
	now := time.Now()
	// Fresh pointer every frame, so hover follows the cursor at full rate.
	var cur POINT
	if r, _, _ := pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur))); r != 0 && cur != a.lastCur {
		a.lastCur = cur
		a.wheel.SetPointer(float64(cur.X-a.winX), float64(cur.Y-a.winY))
		a.dirty = true
	}
	// A soft click each time the pointer moves onto an available option.
	if a.wheel.Revealed() && a.wheel.Phase() == ui.PhaseOpen {
		soundKeepAlive()
		h := a.wheel.EnabledAt(float64(cur.X-a.winX), float64(cur.Y-a.winY))
		if h >= 0 && h != a.lastHover {
			playHover()
		}
		a.lastHover = h
	} else {
		a.lastHover = -1
	}
	for done := false; !done; {
		select {
		case t := <-a.thumbs:
			if len(a.paths) > 0 && t.path == a.paths[0] && t.thumb != nil {
				a.setFileInfo(t.thumb)
			}
		default:
			done = true
		}
	}
	phase := a.wheel.Phase()
	// Safety net: if the button is no longer held and nothing was dropped,
	// close even if the button-up event was missed.
	if phase == ui.PhaseOpen && !a.dropped && a.closeDeadline.IsZero() && !leftButtonDown() && func() bool { h, _ := captureWindow(); return h == 0 }() && now.Sub(a.openedAt) > 150*time.Millisecond {
		a.closeDeadline = now.Add(300 * time.Millisecond)
	}
	// An invisible wheel that no file entered (not a file drag) goes away.
	if !a.wheel.Revealed() && phase == ui.PhaseOpen && now.Sub(a.openedAt) > 3*time.Second {
		logf("closed: no file entered the wheel (not a file drag)")
		a.wheel.Close(now)
	}
	if !a.closeDeadline.IsZero() && now.After(a.closeDeadline) && !a.dropped {
		a.closeDeadline = time.Time{}
		if phase == ui.PhaseOpen {
			logf("closed: released without dropping on a format (files seen: %d)", len(a.paths))
		}
		a.wheel.Close(now)
	}
	if !a.finishedAt.IsZero() && now.Sub(a.finishedAt) > 750*time.Millisecond {
		a.finishedAt = time.Time{}
		a.wheel.Close(now)
	}
	if a.wheel.Closed(now) {
		pShowWindow.Call(a.overlay, SW_HIDE)
		a.wheel = nil
		a.syncFlags()
		if a.pending != nil {
			a.showPanel()
			return
		}
		a.stopTimer()
		return
	}
	a.syncFlags()
	if a.dirty || a.wheel.Animating(now) {
		a.dirty = false
		a.present(a.wheel.Render(now))
	} else if phase == ui.PhaseOpen && a.wheel.Revealed() && len(a.paths) > 0 {
		// Quiet moment: get the other wheel (formats/tools) ready in advance.
		if alt := a.itemsFor(1 - a.mode); !a.wheel.Prebuilt(alt) {
			t := time.Now()
			a.wheel.Prebuild(alt)
			logf("prepared the other wheel in %v", time.Since(t).Round(time.Millisecond))
		}
	}
}

func (a *App) itemsFor(mode int) []ui.Item {
	var items []conv.Item
	if mode == 0 {
		items = conv.FormatItems(a.paths[0])
	} else {
		items = conv.ToolItemsFor(a.paths)
	}
	out := make([]ui.Item, len(items))
	for i, it := range items {
		out[i] = ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled}
	}
	return out
}

// ---------- drag & drop events (called by the COM drop target) ----------

func (a *App) onDragEnter(paths []string, pt POINT) uint32 {
	if a.wheel == nil || len(paths) == 0 {
		return DROPEFFECT_NONE
	}
	if !a.wheel.Revealed() {
		logf("drag entered with %d file(s): %s", len(paths), filepath.Base(paths[0]))
		a.wheel.Reveal(time.Now())
		a.syncFlags()
		playAppear()
	}
	if len(a.paths) == 0 || a.paths[0] != paths[0] {
		a.paths = paths
		a.applyItems(true)
		a.setFileInfo(nil)
		// The center is open: Windows' own drag image shows the file, so no thumbnail is needed.
	}
	return a.onDragOver(pt)
}

func (a *App) setFileInfo(thumb *image.RGBA) {
	first := a.paths[0]
	size := ""
	if len(a.paths) > 1 {
		size = fmt.Sprintf("%d FILES", len(a.paths))
	} else if st, err := os.Stat(first); err == nil {
		size = humanSize(st.Size())
	}
	if conv.Category(first) == "" {
		size = "NOT SUPPORTED"
	}
	a.wheel.SetFileInfo(thumb, conv.ExtLabel(first), size, "")
	a.dirty = true
}

func (a *App) applyItems(animate bool) {
	if a.wheel == nil || len(a.paths) == 0 {
		return
	}
	a.wheel.SetItems(a.itemsFor(a.mode), animate, time.Now())
	a.dirty = true
}

func (a *App) onDragOver(pt POINT) uint32 {
	if a.wheel == nil {
		return DROPEFFECT_NONE
	}
	x, y := float64(pt.X-a.winX), float64(pt.Y-a.winY)
	a.wheel.SetPointer(x, y)
	a.dirty = true
	if a.wheel.EnabledAt(x, y) >= 0 {
		return DROPEFFECT_COPY
	}
	return DROPEFFECT_NONE
}

func (a *App) onDragLeave() {}

func (a *App) onDrop(paths []string, pt POINT) uint32 {
	if a.wheel == nil {
		return DROPEFFECT_NONE
	}
	if len(paths) == 0 {
		paths = a.paths
	}
	x, y := float64(pt.X-a.winX), float64(pt.Y-a.winY)
	idx := a.wheel.EnabledAt(x, y)
	if idx < 0 || len(paths) == 0 {
		logf("dropped outside the formats; cancelled")
		a.closeWheel()
		return DROPEFFECT_NONE
	}
	a.dropped = true
	a.dropPress = ps.pressSeq
	playDrop()
	a.closeDeadline = time.Time{}
	id := a.wheel.Items[idx].ID
	logf("dropped %d file(s) on %s", len(paths), id)
	// Compressing one video asks how first (the dialog opens where the
	// wheel was); everything else starts at once.
	if id == "v:compress" && len(paths) == 1 {
		cx := a.winX + int32(a.wheel.CX)
		cy := a.winY + int32(a.wheel.CY)
		scale := a.wheel.Scale
		label := a.wheel.Items[idx].Label
		a.wheel.CloseFast(time.Now())
		a.setClickThrough(true)
		a.syncFlags()
		// hide the wheel now, so the dialog's glass captures the desktop
		// and not the wheel itself (the progress window does the same)
		pShowWindow.Call(a.overlay, SW_HIDE)
		pDwmFlush.Call()
		a.showCompress(paths[0], id, label, cx, cy, scale)
		return DROPEFFECT_COPY
	}
	a.jobSeq++
	job := a.jobSeq
	jobProgress.Store(job, -1.0)
	label := a.wheel.Items[idx].Label
	// Only files that this option really applies to are processed (a
	// selection can mix kinds; the wheel shows the first file's options).
	paths = applicable(paths, id)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancels[job] = cancel
	// The wheel vanishes quickly; the progress window pops up in its place.
	file := filepath.Base(paths[0])
	if n := countConvertible(paths); n > 1 {
		file = fmt.Sprintf(T("%d files", "%d ملفات"), n)
	}
	a.pending = &panelReq{job: job, title: jobTitle(id, label), file: file,
		cx: a.winX + int32(a.wheel.CX), cy: a.winY + int32(a.wheel.CY), scale: a.wheel.Scale}
	a.wheel.CloseFast(time.Now())
	a.setClickThrough(true)
	a.syncFlags()
	a.dirty = true
	go func() {
		defer cancel()
		res := result{job: job, ok: true, label: label}
		var errs []string
		report := func(f float64) {
			jobProgress.Store(job, f)
			pPostMessageW.Call(a.msgWnd, wmJobProgress, uintptr(job), 0)
		}
		var safeRunPrep func(p string) (string, error)
		safeRunPrep = func(p string) (string, error) {
			engineMu.Lock()
			defer engineMu.Unlock()
			if conv.NeedsResvg(p) && !conv.HaveResvg() {
				logf("downloading resvg")
				report(0)
				if err := conv.EnsureResvg(report); err != nil {
					return "", err
				}
				logf("resvg ready")
			}
			if conv.NeedsPDFium(p, id) && !conv.HavePDFium() {
				logf("downloading PDFium")
				report(0)
				if err := conv.EnsurePDFium(report); err != nil {
					return "", err
				}
				logf("PDFium ready")
			}
			if conv.NeedsFFmpeg(p, id) && !conv.HaveFFmpeg() {
				logf("downloading FFmpeg")
				report(0)
				if err := conv.EnsureFFmpeg(report); err != nil {
					return "", err
				}
				logf("FFmpeg ready")
			}
			if conv.NeedsQPDF(p, id) && !conv.HaveQPDF() {
				logf("downloading qpdf")
				report(0)
				if err := conv.EnsureQPDF(report); err != nil {
					return "", err
				}
				logf("qpdf ready")
			}
			return "", nil
		}
		safeRun := func(p string, i, n int) (out string, err error) {
			defer func() {
				if r := recover(); r != nil {
					logf("panic converting %s: %v", p, r)
					err = errors.New(T("Unexpected error while processing", "خطأ غير متوقع أثناء المعالجة"))
				}
			}()
			if _, err := safeRunPrep(p); err != nil {
				return "", err
			}
			// 100% is shown only when the finished file is in place, so the
			// bar never says done before the file appears.
			report(float64(i) / float64(n))
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return conv.Run(ctx, p, id, func(f float64) {
				report(math.Min((float64(i)+f)/float64(n), 0.99))
			})
		}
		todo := paths
		if len(todo) == 0 {
			errs = append(errs, T("This option doesn't apply to the selected files", "هذا الخيار لا ينطبق على الملفات المحددة"))
		}
		if conv.IsMerge(id) {
			// one output from the whole selection
			func() {
				defer func() {
					if r := recover(); r != nil {
						logf("panic merging: %v", r)
						errs = append(errs, T("Unexpected error while processing", "خطأ غير متوقع أثناء المعالجة"))
					}
				}()
				for _, p := range todo {
					if _, err := safeRunPrep(p); err != nil {
						errs = append(errs, err.Error())
						return
					}
				}
				out, err := conv.Merge(ctx, todo, id, func(f float64) { report(math.Min(f, 0.99)) })
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					logf("merge failed: %v", err)
					errs = append(errs, err.Error())
					return
				}
				res.count = 1
				notifyCreated(out)
			}()
			todo = nil
		}
		for i, p := range todo {
			out, err := safeRun(p, i, len(todo))
			if ctx.Err() != nil {
				break // cancelled: nothing more, no error message
			}
			if err != nil {
				logf("failed %s: %v", filepath.Base(p), err)
				errs = append(errs, filepath.Base(p)+": "+err.Error())
				continue
			}
			res.count++
			notifyCreated(out)
		}
		if len(errs) > 0 {
			res.ok = false
			res.errText = strings.Join(errs, "\n")
		}
		a.results <- res
		pPostMessageW.Call(a.msgWnd, wmJobDone, 0, 0)
	}()
	return DROPEFFECT_COPY
}

// quit stops running conversions (their engines are closed and unfinished
// files removed), then leaves.
func (a *App) quit() {
	if n := len(a.cancels); n > 0 {
		logf("quitting: stopping %d running job(s)", n)
		for job, c := range a.cancels {
			a.cancelled[job] = true
			c()
		}
		deadline := time.Now().Add(4 * time.Second)
		for len(a.cancels) > 0 && time.Now().Before(deadline) {
			select {
			case r := <-a.results:
				delete(a.cancels, r.job)
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	pShowWindow.Call(a.overlay, SW_HIDE)
	pPostQuitMessage.Call(0)
}

func notifyCreated(path string) {
	pSHChangeNotify.Call(SHCNE_CREATE, SHCNF_PATHW, uintptr(unsafe.Pointer(utf16(path))), 0)
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// ---------- drawing to the layered window ----------

func (a *App) present(img *image.RGBA) {
	a.lastPresented = img
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if a.dib == 0 || a.dibW != w || a.dibH != h {
		if a.dib != 0 {
			pDeleteObject.Call(a.dib)
			pDeleteDC.Call(a.memDC)
		}
		screen, _, _ := pGetDC.Call(0)
		a.memDC, _, _ = pCreateCompatibleDC.Call(screen)
		bi := BITMAPINFOHEADER{BiWidth: int32(w), BiHeight: -int32(h), BiPlanes: 1, BiBitCount: 32, BiCompression: BI_RGB}
		bi.BiSize = uint32(unsafe.Sizeof(bi))
		a.dib, _, _ = pCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&a.bits)), 0, 0)
		pSelectObject.Call(a.memDC, a.dib)
		pReleaseDC.Call(0, screen)
		a.dibW, a.dibH = w, h
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(a.bits)), w*h*4)
	src := img.Pix
	for i := 0; i+3 < len(dst); i += 4 {
		dst[i], dst[i+1], dst[i+2], dst[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
	ptDst := POINT{a.winX, a.winY}
	sz := SIZE{int32(w), int32(h)}
	ptSrc := POINT{}
	blend := BLENDFUNCTION{0, 0, 255, AC_SRC_ALPHA}
	screen, _, _ := pGetDC.Call(0)
	r, _, e := pUpdateLayeredWindow.Call(a.overlay, screen, uintptr(unsafe.Pointer(&ptDst)), uintptr(unsafe.Pointer(&sz)),
		a.memDC, uintptr(unsafe.Pointer(&ptSrc)), 0, uintptr(unsafe.Pointer(&blend)), ULW_ALPHA)
	pReleaseDC.Call(0, screen)
	if r == 0 {
		logf("UpdateLayeredWindow failed: %v", e)
	}
}

func captureScreen(x, y, w, h int32) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	screen, _, _ := pGetDC.Call(0)
	defer pReleaseDC.Call(0, screen)
	mem, _, _ := pCreateCompatibleDC.Call(screen)
	defer pDeleteDC.Call(mem)
	bi := BITMAPINFOHEADER{BiWidth: w, BiHeight: -h, BiPlanes: 1, BiBitCount: 32, BiCompression: BI_RGB}
	bi.BiSize = uint32(unsafe.Sizeof(bi))
	var bits uintptr
	hbm, _, _ := pCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 {
		return out
	}
	defer pDeleteObject.Call(hbm)
	old, _, _ := pSelectObject.Call(mem, hbm)
	pBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(x), uintptr(y), SRCCOPY)
	pSelectObject.Call(mem, old)
	src := unsafe.Slice((*byte)(unsafe.Pointer(bits)), int(w*h*4))
	for i := 0; i+3 < len(src); i += 4 {
		out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = src[i+2], src[i+1], src[i], 255
	}
	return out
}

// ---------- settings ----------

func isDarkTheme() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v == 0
}

func firstRun() bool {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS)
	if err != nil {
		return false
	}
	defer k.Close()
	if _, _, err := k.GetIntegerValue("Welcomed"); err == nil {
		return false
	}
	k.SetDWordValue("Welcomed", 1)
	return true
}

// warmUp renders an off-screen wheel once at startup so fonts, glyphs and
// code paths are ready and the first real opening is instant.
func warmUp(fonts *ui.Fonts) {
	t := time.Now()
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		n := int(ui.LogicalSize * scale)
		bg := image.NewRGBA(image.Rect(0, 0, n, n))
		w := ui.New(scale, n, n, float64(n)/2, float64(n)/2, ui.FrostBackground(bg, 26*scale, 1.4), false, fonts, t)
		w.Reveal(t)
		var items []ui.Item
		for _, it := range append(conv.FormatItems("x.png"), conv.ToolItems("x.png")...) {
			items = append(items, ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled})
		}
		w.SetItems(items, false, t)
		w.SetFileInfo(nil, "PNG", "1.0 MB", "NOT SUPPORTED YET")
		w.Render(t.Add(time.Second))
	}
	logf("warm-up done in %v", time.Since(t).Round(time.Millisecond))
}

func jobProgressOf(job int) float64 {
	if v, ok := jobProgress.Load(job); ok {
		return v.(float64)
	}
	return -1
}

func jobProgressText(job int) string {
	if v := jobProgressOf(job); v >= 0 {
		return fmt.Sprintf("%d%%", int(v*100))
	}
	return ""
}

// handleResults finishes jobs: on the wheel if it still shows that job,
// otherwise with a notification.
func (a *App) handleResults() {
	for {
		select {
		case r := <-a.results:
			logf("job %d finished: ok=%v files=%d %s", r.job, r.ok, r.count, r.errText)
			delete(a.cancels, r.job)
			jobProgress.Delete(r.job)
			if a.cancelled[r.job] {
				delete(a.cancelled, r.job)
				continue
			}
			shown := a.panel != nil && r.job == a.panelJob
			waiting := a.pending != nil && r.job == a.pending.job
			if shown || waiting {
				// done: the window disappears at once, no animation
				if waiting {
					a.pending = nil
				}
				a.hidePanel()
				a.setTrayTip(idleTip())
				if !r.ok {
					a.showFailure(r.errText)
				}
				continue
			}
			if a.background > 0 {
				a.background--
			}
			a.setTrayTip(idleTip())
			if r.ok {
				a.balloon(T("Done", "اكتمل التحويل"), fmt.Sprintf(T("%s — %d file(s)", "%s — %d ملف"), r.label, r.count), false)
			} else {
				a.showFailure(r.errText)
			}
		default:
			return
		}
	}
}

// showFailure tells the user a job failed. A tray notification alone can be
// silenced by Windows (Do not disturb, notifications off), which made a
// failed job look like the progress window just vanished, so a message box
// is shown as well. It runs on its own goroutine so the wheel, the tray and
// other jobs keep working while it is open.
func (a *App) showFailure(errText string) {
	title := T("Conversion failed", "تعذّر التحويل")
	a.balloon(title, errText, true)
	go messageBox(title+"\n\n"+errText, MB_ICONERROR)
}

// applicable keeps the files for which the option id exists and is enabled.
func applicable(paths []string, id string) []string {
	var out []string
	for _, p := range paths {
		if conv.Category(p) == "" {
			continue
		}
		if conv.IsMerge(id) {
			out = append(out, p) // Merge picks the right kind itself
			continue
		}
		var items []conv.Item
		if strings.HasPrefix(id, "to:") {
			items = conv.FormatItems(p)
		} else {
			items = conv.ToolItemsFor([]string{p})
		}
		for _, it := range items {
			if it.ID == id && it.Enabled {
				out = append(out, p)
				break
			}
		}
		if len(out) == 0 || out[len(out)-1] != p {
			logf("skipped %s: %s does not apply to it", filepath.Base(p), id)
		}
	}
	return out
}

func countConvertible(paths []string) int {
	n := 0
	for _, p := range paths {
		if conv.Category(p) != "" {
			n++
		}
	}
	return n
}

// jobTitle is the panel's top line: "Converting to MP3", or the tool's action.
func jobTitle(id, label string) string {
	if strings.HasPrefix(id, "to:") {
		return fmt.Sprintf(T("Converting to %s", "جارٍ التحويل إلى %s"), label)
	}
	tool := id
	if k := strings.IndexByte(id, ':'); k >= 0 {
		tool = id[k+1:]
	}
	switch tool {
	case "compress":
		return T("Compressing", "جارٍ الضغط")
	case "metadata":
		return T("Removing metadata", "جارٍ إزالة البيانات الوصفية")
	case "rotate":
		return T("Rotating", "جارٍ التدوير")
	case "resize":
		return T("Resizing to 50%", "جارٍ التصغير إلى 50%")
	case "addbg":
		return T("Adding a white background", "جارٍ إضافة خلفية بيضاء")
	case "mute":
		return T("Removing the sound", "جارٍ إزالة الصوت")
	case "speed":
		return T("Speeding up 2×", "جارٍ التسريع 2×")
	case "snapshot":
		return T("Taking a snapshot", "جارٍ التقاط صورة")
	case "split":
		return T("Splitting", "جارٍ التقسيم")
	case "normalize":
		return T("Normalizing the volume", "جارٍ موازنة الصوت")
	case "visualize":
		return T("Drawing the waveform", "جارٍ رسم الموجة الصوتية")
	case "channels":
		return T("Changing channels", "جارٍ تغيير القنوات")
	case "extract":
		return T("Extracting", "جارٍ الاستخراج")
	case "merge":
		return T("Merging into one PDF", "جارٍ الدمج في ملف PDF واحد")
	case "blackbg":
		return T("Adding a black background", "جارٍ إضافة خلفية سوداء")
	}
	return label
}

// startPump drives animation frames in step with the display: DwmFlush
// returns once per compositor refresh (60, 120, 144 Hz ...), so the wheel
// draws exactly one frame per screen refresh. Without composition it falls
// back to a ~120 fps clock.
func (a *App) startPump() {
	stop := make(chan struct{})
	a.pumpStop = stop
	hwnd := a.overlay
	go func() {
		lastPost := time.Now()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if r, _, _ := pDwmFlush.Call(); r != 0 {
				time.Sleep(8 * time.Millisecond)
			}
			if !a.framePending.Swap(true) {
				pPostMessageW.Call(hwnd, wmFrame, 0, 0)
				lastPost = time.Now()
			} else if time.Since(lastPost) > 250*time.Millisecond {
				// a frame message got lost: never let the animation freeze
				pPostMessageW.Call(hwnd, wmFrame, 0, 0)
				lastPost = time.Now()
			}
		}
	}()
}

// winrarFromRegistry finds Rar.exe next to WinRAR.exe wherever it was
// installed (App Paths is written by WinRAR's installer).
func winrarFromRegistry() string {
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, `Software\Microsoft\Windows\CurrentVersion\App Paths\WinRAR.exe`, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue("")
		k.Close()
		if err == nil && v != "" {
			return filepath.Join(filepath.Dir(strings.Trim(v, `"`)), "Rar.exe")
		}
	}
	return ""
}

// animatePanelClose plays the closing shrink-and-fade of the last frame
// (ease-in, about 160 ms) before the window hides.
func (a *App) animatePanelClose() {
	img := a.lastPresented
	if img == nil {
		return
	}
	for i := 1; i <= 10; i++ {
		e := float64(i) / 10
		e *= e
		a.present(ui.ZoomFrame(img, 1-0.08*e, 1-e))
		pDwmFlush.Call()
	}
}
