//go:build windows

package main

// The video windows (Trim Video, Video Snapshots; later Split, Speed,
// Crop): our
// own layered window like the Compress dialog, opening where the wheel was.
// Frames and thumbnails come from FFmpeg on other goroutines; they hand
// their results over under vwin.mu and post wmVideoUpdate, and only the
// window's own thread touches the dialog. No sound is played here.

import (
	"context"
	"errors"
	"image"
	"math"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"raha/conv"
	"raha/ui"
)

const (
	wmVideoUpdate = WM_APP + 40
	wmVideoSaved  = WM_APP + 41 // a snapshot was saved (wParam: the window's gen)
	savedTimer    = 51          // hides the "Saved" line
	savedMs       = 2000
	VK_LEFT       = 0x25
	VK_RIGHT      = 0x27
)

// the kinds of video window
const (
	vkTrim = iota + 1
	vkSnap
)

var vwin struct {
	wnd      uintptr
	kind     int
	trim     *ui.TrimDialog
	snap     *ui.SnapDialog
	path     string
	id       string // the wheel item ("v:trim")
	label    string // its wheel label, for the job's title and results
	info     conv.VideoInfo
	x, y     int32
	shown    bool
	drag     bool
	dragOff  POINT
	tracking bool
	last     *image.RGBA
	cancel   context.CancelFunc
	req      chan float64 // the next frame to show (latest wins)

	mu       sync.Mutex // guards the hand-over below
	gen      int        // bumped on every open/close: late results are dropped
	upReady  bool       // info arrived (or failed)
	upFailed bool
	upInfo   conv.VideoInfo
	upFrame  *image.RGBA
	upThumbs map[int]*image.RGBA
	upProg   float64
}

// showVideoWindow opens the window for id ("v:trim") on path, centred on
// (cx, cy) and kept inside that monitor's work area.
func (a *App) showVideoWindow(path, id, label string, cx, cy int32, scale float64) {
	if vwin.shown {
		hideVideoWindow()
	}
	if vwin.wnd == 0 {
		cls := utf16("FalakVideo")
		arrow, _, _ := pLoadCursorW.Call(0, 32512)
		wc := WNDCLASSEXW{LpfnWndProc: syscall.NewCallback(videoProc), HInstance: a.hInst, LpszClassName: cls, HCursor: arrow}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		vwin.wnd, _, _ = pCreateWindowExW.Call(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
			uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("Falak video"))), WS_POPUP,
			0, 0, 10, 10, 0, 0, a.hInst, 0)
		logAffinity("video window", vwin.wnd)
	}
	vwin.path, vwin.id, vwin.label = path, id, label
	vwin.info = conv.VideoInfo{}
	vwin.trim, vwin.snap = nil, nil
	var w, h, tw, th, pw, ph int
	switch id {
	case "v:trim":
		vwin.kind = vkTrim
		vwin.trim = ui.NewTrimDialog(scale, a.fonts, darkMode(), arabic, ui.TrimLabels{
			Title:         T("Trim Video", "قص الفيديو"),
			File:          filepath.Base(path),
			Hint:          T("Drag the ends to trim", "اسحب الطرفين لتحديد المقطع"),
			Selected:      T("%s selected", "المدة المحددة: %s"),
			Trim:          T("Trim", "قص"),
			Preparing:     T("Preparing the video…", "جار تجهيز الفيديو…"),
			PreviewFailed: T("Couldn't show the preview", "تعذّر عرض المعاينة"),
		})
		w, h = vwin.trim.Size()
		tw, th = vwin.trim.ThumbSize()
		pw, ph = vwin.trim.PreviewSize()
	case "v:snapshot":
		vwin.kind = vkSnap
		vwin.snap = ui.NewSnapDialog(scale, a.fonts, darkMode(), arabic, ui.SnapLabels{
			Title:         T("Video Snapshots", "لقطات من الفيديو"),
			File:          filepath.Base(path),
			Save:          T("Save Snapshot", "حفظ اللقطة"),
			Saved:         T("Saved", "تم الحفظ"),
			Preparing:     T("Preparing the video…", "جار تجهيز الفيديو…"),
			PreviewFailed: T("Couldn't show the preview", "تعذّر عرض المعاينة"),
		})
		w, h = vwin.snap.Size()
		pw, ph = vwin.snap.PreviewSize() // no thumbnails (tw, th stay 0)
	default:
		return
	}
	var pt POINT
	pt.X, pt.Y = cx, cy
	mon, _, _ := pMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), MONITOR_DEFAULTTONEAREST)
	var mi MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	vwin.x = clamp32(cx-int32(w)/2, mi.RcWork.Left, mi.RcWork.Right-int32(w))
	vwin.y = clamp32(cy-int32(h)/2, mi.RcWork.Top, mi.RcWork.Bottom-int32(h))

	ctx, cancel := context.WithCancel(context.Background())
	vwin.cancel = cancel
	vwin.req = make(chan float64, 1)
	vwin.mu.Lock()
	vwin.gen++
	gen := vwin.gen
	vwin.upReady, vwin.upFailed, vwin.upFrame, vwin.upThumbs, vwin.upProg = false, false, nil, map[int]*image.RGBA{}, -1
	vwin.mu.Unlock()

	refreshVideoWindow()
	pShowWindow.Call(vwin.wnd, SW_SHOW)
	pSetForegroundWindow.Call(vwin.wnd)
	vwin.shown = true
	logf("video window %s shown at %d,%d (%dx%d) for %s", id, vwin.x, vwin.y, w, h, filepath.Base(path))
	go videoWork(ctx, gen, path, vwin.req, pw, ph, tw, th)
}

// videoWork gets FFmpeg if needed (the same download as every video job,
// under engineMu), reads the video's length, then loads the thumbnails and
// serves frame requests (pw×ph previews, tw×th thumbnails) until the
// window closes.
func videoWork(ctx context.Context, gen int, path string, req chan float64, pw, ph, tw, th int) {
	hand := func(f func()) bool { // hand a result to the window
		vwin.mu.Lock()
		ok := vwin.gen == gen
		if ok {
			f()
		}
		vwin.mu.Unlock()
		if ok {
			pPostMessageW.Call(vwin.wnd, wmVideoUpdate, 0, 0)
		}
		return ok
	}
	fail := func(err error) {
		logf("video window: %v", err)
		hand(func() { vwin.upReady, vwin.upFailed = true, true })
	}
	if err := func() error {
		engineMu.Lock()
		defer engineMu.Unlock()
		if conv.HaveFFmpeg() {
			return nil
		}
		logf("downloading FFmpeg (video window)")
		hand(func() { vwin.upProg = 0 })
		if err := conv.EnsureFFmpeg(func(f float64) { hand(func() { vwin.upProg = f }) }); err != nil {
			return err
		}
		logf("FFmpeg ready")
		return nil
	}(); err != nil {
		fail(err)
		return
	}
	info := conv.ProbeVideo(path)
	if info.Duration <= 0 {
		fail(errors.New("the video's length couldn't be read"))
		return
	}
	if !hand(func() { vwin.upInfo, vwin.upReady, vwin.upProg = info, true, -1 }) {
		return
	}
	if tw > 0 {
		go conv.Thumbs(ctx, path, info.Duration, ui.StripCells, tw, th, func(i int, img *image.RGBA) {
			hand(func() { vwin.upThumbs[i] = img })
		})
	}
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-req:
			img, err := conv.FrameAt(ctx, path, t, pw, ph, true)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				logf("video window: frame at %.3f: %v", t, err)
				continue
			}
			hand(func() { vwin.upFrame = img })
		}
	}
}

// requestFrame asks for the frame at seek position t; an older request
// that has not started yet is dropped.
func requestFrame(t float64) {
	select {
	case <-vwin.req:
	default:
	}
	select {
	case vwin.req <- t:
	default:
	}
}

// takeVideoUpdates moves what the worker handed over into the dialog.
func takeVideoUpdates() {
	vwin.mu.Lock()
	ready, failed, info, frame, prog := vwin.upReady, vwin.upFailed, vwin.upInfo, vwin.upFrame, vwin.upProg
	thumbs := vwin.upThumbs
	vwin.upFrame, vwin.upThumbs = nil, map[int]*image.RGBA{}
	vwin.mu.Unlock()
	if d := vwin.trim; d != nil {
		d.Progress = prog
		if failed {
			d.Failed = true
		}
		if ready && !failed && d.Duration == 0 {
			vwin.info = info
			d.SetDuration(info.Duration, 1/info.FrameRate())
			requestFrame(0)
		}
		if frame != nil {
			d.Frame = frame
		}
		for i, t := range thumbs {
			if i >= 0 && i < len(d.Thumbs) {
				d.Thumbs[i] = t
			}
		}
	}
	if d := vwin.snap; d != nil {
		d.Progress = prog
		if failed {
			d.Failed = true
		}
		if ready && !failed && d.Duration == 0 {
			vwin.info = info
			d.SetVideo(info.Duration, info.FrameRate())
			requestFrame(info.FrameSeek(0))
		}
		if frame != nil {
			d.Frame = frame
		}
	}
}

func hideVideoWindow() {
	if !vwin.shown {
		return
	}
	vwin.shown = false
	pKillTimer.Call(vwin.wnd, savedTimer)
	if vwin.cancel != nil {
		vwin.cancel()
	}
	vwin.mu.Lock()
	vwin.gen++
	vwin.mu.Unlock()
	if vwin.drag {
		vwin.drag = false
		pReleaseCapture.Call()
	}
	if vwin.last != nil {
		pDwmFlush.Call()
		animateLayeredClose(vwin.wnd, vwin.last, vwin.x, vwin.y)
	}
	pShowWindow.Call(vwin.wnd, SW_HIDE)
}

func refreshVideoWindow() {
	var img *image.RGBA
	switch {
	case vwin.trim != nil:
		img = vwin.trim.Render()
	case vwin.snap != nil:
		img = vwin.snap.Render()
	default:
		return
	}
	vwin.last = img
	presentLayered(vwin.wnd, img, vwin.x, vwin.y)
}

func videoProc(hwnd, msg, wParam, lParam uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logf("panic in video window: %v", r)
			hideVideoWindow()
			ret = 0
		}
	}()
	x, y := float64(int16(lParam&0xFFFF)), float64(int16((lParam>>16)&0xFFFF))
	switch msg {
	case wmVideoUpdate:
		if vwin.shown {
			takeVideoUpdates()
			refreshVideoWindow()
		}
		return 0
	case wmVideoSaved:
		vwin.mu.Lock()
		same := int(wParam) == vwin.gen
		vwin.mu.Unlock()
		if d := vwin.snap; d != nil && same && vwin.shown {
			d.Saved = true
			pSetTimer.Call(hwnd, savedTimer, savedMs, 0)
			refreshVideoWindow()
		}
		return 0
	case WM_TIMER:
		if wParam == savedTimer {
			pKillTimer.Call(hwnd, savedTimer)
			if d := vwin.snap; d != nil {
				d.Saved = false
				refreshVideoWindow()
			}
		}
		return 0
	case WM_KEYDOWN:
		switch wParam {
		case VK_ESCAPE:
			hideVideoWindow()
		case vkReturn:
			app.videoAction()
		case VK_LEFT, VK_RIGHT: // a frame towards that side of the scrubber
			if d := vwin.snap; d != nil && d.Ready() {
				n := 1
				if (wParam == VK_LEFT) != d.RTL {
					n = -1
				}
				snapShow(d, d.Pos+n)
			}
		}
		return 0
	case WM_LBUTTONDOWN:
		if d := vwin.snap; d != nil {
			switch d.HitTest(x, y) {
			case ui.SnapSlider:
				d.Dragging = true
				snapShow(d, d.FrameAtX(x))
				pSetCapture.Call(hwnd)
			case ui.SnapNone:
				vwin.drag = true
				vwin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
			return 0
		}
		if d := vwin.trim; d != nil {
			switch h := d.HitTest(x, y); h {
			case ui.TrimStartHandle, ui.TrimEndHandle:
				d.Active = h
				pSetCapture.Call(hwnd)
				return 0
			case ui.TrimStrip: // the nearer handle jumps here
				d.Active = d.Nearest(x)
				trimMoveTo(d, x)
				pSetCapture.Call(hwnd)
				return 0
			case ui.TrimNone:
				vwin.drag = true
				vwin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
		}
		return 0
	case WM_MOUSEMOVE:
		if vwin.drag {
			var cur POINT
			pGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
			nx, ny := cur.X-vwin.dragOff.X, cur.Y-vwin.dragOff.Y
			if nx != vwin.x || ny != vwin.y {
				vwin.x, vwin.y = nx, ny
				presentLayered(vwin.wnd, vwin.last, vwin.x, vwin.y)
			}
			return 0
		}
		if d := vwin.trim; d != nil && d.Active != 0 {
			trimMoveTo(d, x)
			return 0
		}
		if d := vwin.snap; d != nil && d.Dragging {
			snapShow(d, d.FrameAtX(x))
			return 0
		}
		if !vwin.tracking {
			tme := trackMouseEvent{DwFlags: TME_LEAVE, HwndTrack: hwnd}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			vwin.tracking = true
		}
		return 0
	case WM_MOUSELEAVE:
		vwin.tracking = false
		return 0
	case 0x0215: // WM_CAPTURECHANGED: any drag has ended
		vwin.drag = false
		if d := vwin.trim; d != nil && d.Active != 0 {
			d.Active = 0
			refreshVideoWindow()
		}
		if d := vwin.snap; d != nil && d.Dragging {
			d.Dragging = false
		}
		return 0
	case WM_LBUTTONUP:
		if vwin.drag {
			vwin.drag = false
			pReleaseCapture.Call()
			return 0
		}
		if d := vwin.snap; d != nil {
			if d.Dragging {
				d.Dragging = false
				pReleaseCapture.Call()
				return 0
			}
			switch d.HitTest(x, y) {
			case ui.SnapClose:
				hideVideoWindow()
			case ui.SnapPrev:
				snapShow(d, d.Pos-1)
			case ui.SnapNext:
				snapShow(d, d.Pos+1)
			case ui.SnapSave:
				app.videoAction()
			}
			return 0
		}
		if d := vwin.trim; d != nil {
			if d.Active != 0 {
				d.Active = 0
				pReleaseCapture.Call()
				refreshVideoWindow()
				return 0
			}
			switch d.HitTest(x, y) {
			case ui.TrimClose:
				hideVideoWindow()
			case ui.TrimButton:
				app.videoAction()
			}
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// trimMoveTo moves the dragged handle to window pixel x and shows the
// frame under it (for the end handle: the last frame kept).
func trimMoveTo(d *ui.TrimDialog, x float64) {
	t := d.Move(d.Active, d.TimeAt(x))
	d.Error = ""
	fr := vwin.info.FrameRate()
	k := int(math.Round(t * fr))
	if d.Active == ui.TrimEndHandle {
		k = max(0, k-1)
	}
	requestFrame(vwin.info.FrameSeek(k))
	refreshVideoWindow()
}

// snapShow moves the Snapshots window to frame k and asks for its picture.
func snapShow(d *ui.SnapDialog, k int) {
	if d.Step(k - d.Pos) {
		requestFrame(vwin.info.FrameSeek(d.Pos))
	}
	refreshVideoWindow()
}

// videoAction is Enter or the window's button.
func (a *App) videoAction() {
	if d := vwin.trim; d != nil {
		a.trimStart(d)
	}
	if d := vwin.snap; d != nil {
		a.snapSave(d)
	}
}

// snapSave saves the frame shown as a PNG beside the video; the window
// stays open, and "Saved" shows once the file is there. Several saves may
// run at once; each gets its own name. Closing the window doesn't stop a
// save that has started.
func (a *App) snapSave(d *ui.SnapDialog) {
	if !d.Ready() {
		return
	}
	path, t := vwin.path, vwin.info.FrameSeek(d.Pos)
	vwin.mu.Lock()
	gen := vwin.gen
	vwin.mu.Unlock()
	wnd := vwin.wnd
	logf("snapshot of %s at frame %d (%.3f s)", filepath.Base(path), d.Pos, t)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logf("panic saving a snapshot of %s: %v", path, r)
			}
		}()
		out, err := conv.SnapshotAt(context.Background(), path, t)
		if err != nil {
			logf("snapshot failed: %v", err)
			a.showError(T("Couldn't save the snapshot", "تعذّر حفظ اللقطة"), filepath.Base(path)+": "+err.Error())
			return
		}
		logf("snapshot saved: %s", out)
		notifyCreated(out)
		pPostMessageW.Call(wnd, wmVideoSaved, uintptr(gen), 0)
	}()
}

// trimStart checks the selection, then closes the window and starts the
// job (or shows the error line and stays open).
func (a *App) trimStart(d *ui.TrimDialog) {
	if !d.Ready() {
		return
	}
	if d.End-d.Start < conv.MinTrim {
		d.Error = T("The selection is too short", "المقطع المحدد قصير جدًا")
		refreshVideoWindow()
		return
	}
	path, start, end := vwin.path, d.Start, d.End
	cx := vwin.x + int32(float64(d.W)/2)
	cy := vwin.y + int32(float64(d.H)/2)
	scale := d.Scale
	hideVideoWindow()
	logf("trim %s: %.3f-%.3f s", filepath.Base(path), start, end)
	a.launchVideoJob(path, vwin.id, vwin.label, cx, cy, scale, func(ctx context.Context, report func(float64)) (string, error) {
		return conv.TrimVideo(ctx, path, start, end, report)
	})
}
