//go:build windows

package main

// The compress dialog: our own layered window (ui.CompressDialog) that
// opens where the wheel was, so the video is compressed the way the user
// asked before anything runs.

import (
	"context"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"raha/conv"
	"raha/ui"
)

const WM_CHAR = 0x0102

var cdlgWin struct {
	wnd      uintptr
	dlg      *ui.CompressDialog
	last     *image.RGBA
	path     string
	duration float64
	x, y     int32
	shown    bool
	tracking bool
	drag     bool
	dragOff  POINT
}

// showCompress opens the dialog centred on (cx, cy), clamped to the work
// area of that monitor.
func (a *App) showCompress(path, id, label string, cx, cy int32, scale float64) {
	if cdlgWin.wnd == 0 {
		cls := utf16("FalakCompress")
		arrow, _, _ := pLoadCursorW.Call(0, 32512)
		wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(compressProc), HInstance: a.hInst, LpszClassName: cls, HCursor: arrow}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		cdlgWin.wnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
			uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("Falak compress"))), WS_POPUP,
			0, 0, 10, 10, 0, 0, a.hInst, 0)

		logAffinity("compress dialog", cdlgWin.wnd)
	}
	cdlgWin.path = path
	cdlgWin.duration = conv.VideoDuration(path)
	sizeText := ""
	var orig int64
	if st, err := os.Stat(path); err == nil {
		sizeText = humanSize(st.Size())
		orig = st.Size()
	}
	labels := ui.CompressLabels{
		Title:        T("Compress", "ضغط"),
		File:         filepath.Base(path),
		SizeLine:     T("Original size:", "الحجم الأصلي:"),
		Preset:       T("Compression level", "مستوى الضغط"),
		Balanced:     T("Balanced", "متوازن"),
		Strong:       T("Strong", "قوي"),
		BalancedHint: T("Smaller with less quality loss.", "حجم أصغر مع فقدان أقل للجودة."),
		StrongHint:   T("Smallest possible size, with clearer quality loss.", "أصغر حجم ممكن، مع فقدان أوضح للجودة."),
		Target:       T("Compress to a set size", "الضغط إلى حجم محدد"),
		TargetLabel:  T("Target size (MB)", "الحجم المطلوب (ميغابايت)"),
		Start:        T("Compress", "ضغط"),
		TargetTooBig: T("Enter a size smaller than the original file", "اكتب حجمًا أصغر من حجم الملف الأصلي"),
	}
	w, h := ui.NewCompressDialog(scale, nil, a.fonts, darkMode(), arabic, labels, sizeText).Size()
	var pt POINT
	pt.X, pt.Y = cx, cy
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	cdlgWin.x = clamp32(cx-int32(w)/2, mi.RcWork.Left, mi.RcWork.Right-int32(w))
	cdlgWin.y = clamp32(cy-int32(h)/2, mi.RcWork.Top, mi.RcWork.Bottom-int32(h))
	cdlgWin.dlg = ui.NewCompressDialog(scale, nil, a.fonts, darkMode(), arabic, labels, sizeText)
	cdlgWin.dlg.OriginalBytes = float64(orig)
	a.refreshCompress()
	pShowWindow.Call(cdlgWin.wnd, SW_SHOW)
	pSetForegroundWindow.Call(cdlgWin.wnd)
	cdlgWin.shown = true
	logf("compress dialog shown at %d,%d (%dx%d) for %s", cdlgWin.x, cdlgWin.y, w, h, filepath.Base(path))
}

func hideCompress() {
	if cdlgWin.shown {
		cdlgWin.shown = false
		if cdlgWin.last != nil {
			pDwmFlush.Call()
			animateLayeredClose(cdlgWin.wnd, cdlgWin.last, cdlgWin.x, cdlgWin.y)
		}
		pShowWindow.Call(cdlgWin.wnd, SW_HIDE)
	}
}

func (a *App) refreshCompress() {
	img := cdlgWin.dlg.Render()
	cdlgWin.last = img
	presentLayered(cdlgWin.wnd, img, cdlgWin.x, cdlgWin.y)
}

func compressProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic in compress dialog: %v", r)
			hideCompress()
			ret = 0
		}
	}()
	d := cdlgWin.dlg
	switch msg {
	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			hideCompress()
		}
		return 0
	case WM_CHAR:
		if d != nil {
			d.Char(rune(wParam))
			app.refreshCompress()
		}
		return 0
	case WM_LBUTTONDOWN:
		if d != nil {
			x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
			if d.HitTest(x, y) == 0 { // glass only: never on a control
				cdlgWin.drag = true
				cdlgWin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
		}
		return 0
	case WM_MOUSEMOVE:
		if cdlgWin.drag {
			var cur POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
			nx, ny := cur.X-cdlgWin.dragOff.X, cur.Y-cdlgWin.dragOff.Y
			if nx != cdlgWin.x || ny != cdlgWin.y {
				cdlgWin.x, cdlgWin.y = nx, ny
				presentLayered(cdlgWin.wnd, cdlgWin.last, cdlgWin.x, cdlgWin.y)
			}
			return 0
		}
		if !cdlgWin.tracking {
			tme := trackMouseEvent{DwFlags: TME_LEAVE, HwndTrack: hwnd}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			cdlgWin.tracking = true
		}
		return 0
	case WM_MOUSELEAVE:
		cdlgWin.tracking = false
		return 0
	case 0x0215: // WM_CAPTURECHANGED: the drag has ended
		cdlgWin.drag = false
		return 0
	case WM_LBUTTONUP:
		if cdlgWin.drag {
			cdlgWin.drag = false
			pReleaseCapture.Call()
			app.refreshCompress()
			return 0
		}
		if d != nil {
			x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
			switch d.HitTest(x, y) {
			case 1:
				hideCompress()
			case 2:
				d.Preset, d.Error = 0, ""
				app.refreshCompress()
			case 3:
				d.Preset, d.Error = 1, ""
				app.refreshCompress()
			case 4:
				d.Target = !d.Target
				if d.Target && d.TargetMB == "" {
					d.TargetMB = "100" // a real value, not a placeholder
				}
				d.Error = ""
				app.refreshCompress()
			case 6:
				app.compressStart()
			}
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// compressStart checks the target size against the video's length, then
// starts the job (or shows the error line and stays).
func (a *App) compressStart() {
	d := cdlgWin.dlg
	if d == nil || !d.StartActive() {
		return
	}
	plan := conv.CompressPlan{Strong: d.Preset == 1}
	if d.Target {
		mb := d.MB()
		kbits := mb*8388.608/cdlgWin.duration - 128
		if cdlgWin.duration <= 0 || kbits < 64 {
			d.Error = T("The target size is too small for this video", "الحجم المطلوب صغير جدًا لهذا الفيديو.")
			a.refreshCompress()
			return
		}
		plan.TargetMB = mb
	}
	cx := cdlgWin.x + int32(float64(d.W)/2)
	cy := cdlgWin.y + int32(float64(d.H)/2)
	scale, label := d.Scale, d.Labels.Start
	hideCompress()
	a.launchCompress(cdlgWin.path, plan, label, cx, cy, scale)
}

// launchCompress starts the job exactly like a drop does: same progress
// window, same results channel, same notifications.
func (a *App) launchCompress(path string, plan conv.CompressPlan, label string, cx, cy int32, scale float64) {
	a.jobSeq++
	job := a.jobSeq
	jobProgress.Store(job, -1.0)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancels[job] = cancel
	a.pending = &panelReq{job: job, title: jobTitle("v:compress", label), file: filepath.Base(path),
		cx: cx, cy: cy, scale: scale}
	a.setClickThrough(true)
	a.syncFlags()
	go func() {
		defer cancel()
		res := result{job: job, ok: true, label: label}
		report := func(f float64) {
			jobProgress.Store(job, f)
			pPostMessageW.Call(a.msgWnd, wmJobProgress, uintptr(job), 0)
		}
		out, err := func() (out string, err error) {
			defer func() {
				if r := recover(); r != nil {
					logf("panic compressing %s: %v", path, r)
					err = errors.New(T("Unexpected error while processing", "خطأ غير متوقع أثناء المعالجة"))
				}
			}()
			engineMu.Lock()
			defer engineMu.Unlock()
			if !conv.HaveFFmpeg() {
				logf("downloading FFmpeg")
				report(0)
				if e := conv.EnsureFFmpeg(report); e != nil {
					return "", e
				}
				logf("FFmpeg ready")
			}
			report(0)
			return conv.RunWithPlan(ctx, path, "v:compress", plan, func(f float64) {
				report(math.Min(f, 0.99))
			})
		}()
		if err != nil {
			res.ok = false
			res.errText = filepath.Base(path) + ": " + err.Error()
		} else {
			res.count = 1
			notifyCreated(out)
		}
		a.results <- res
		pPostMessageW.Call(a.msgWnd, wmJobDone, 0, 0)
	}()
	a.showPanel()
}
