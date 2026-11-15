//go:build windows

package main

// The video windows (Trim Video, Video Snapshots, Split Video, Change
// Video Speed; later Crop): our
// own layered window like the Compress dialog, opening where the wheel was.
// Frames and thumbnails come from FFmpeg on other goroutines; they hand
// their results over under vwin.mu and post wmVideoUpdate, and only the
// window's own thread touches the dialog. No sound is played here.

import (
	"context"
	"errors"
	"fmt"
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
	VK_DELETE     = 0x2E
)

// the kinds of video window
const (
	vkTrim = iota + 1
	vkSnap
	vkSplit
	vkSpeed
)

var vwin struct {
	wnd      uintptr
	kind     int
	trim     *ui.TrimDialog
	snap     *ui.SnapDialog
	split    *ui.SplitDialog
	press    [2]int // what a press in the Split window landed on (kind, point)
	speed    *ui.SpeedDialog
	ctx      context.Context    // ends when the window closes
	play     context.CancelFunc // stops the Speed window's playing preview
	playAt   float64            // where in the video the preview last was
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
	upAt     float64 // the video time of upFrame (the Speed preview)
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
	vwin.trim, vwin.snap, vwin.split, vwin.speed = nil, nil, nil, nil
	vwin.play, vwin.playAt = nil, 0
	vwin.press = [2]int{}
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
	case "v:split":
		vwin.kind = vkSplit
		vwin.split = ui.NewSplitDialog(scale, a.fonts, darkMode(), arabic, ui.SplitLabels{
			Title:         T("Split Video", "تقسيم الفيديو"),
			File:          filepath.Base(path),
			Add:           T("Add Split", "إضافة نقطة تقسيم"),
			Split:         T("Split", "تقسيم"),
			Preparing:     T("Preparing the video…", "جار تجهيز الفيديو…"),
			PreviewFailed: T("Couldn't show the preview", "تعذّر عرض المعاينة"),
			Count:         func(n int) string { return splitCountText(n, arabic) },
		})
		w, h = vwin.split.Size()
		tw, th = vwin.split.ThumbSize()
		pw, ph = vwin.split.PreviewSize()
	case "v:speed":
		vwin.kind = vkSpeed
		chips := make([]string, len(conv.Speeds))
		sel := 0
		for i, s := range conv.Speeds {
			chips[i] = conv.SpeedText(s) + "×"
			if s == 2 { // the old SPEED 2× is where the choice starts
				sel = i
			}
		}
		vwin.speed = ui.NewSpeedDialog(scale, a.fonts, darkMode(), arabic, ui.SpeedLabels{
			Title:         T("Change Video Speed", "تغيير سرعة الفيديو"),
			File:          filepath.Base(path),
			Speed:         T("Speed", "السرعة"),
			Change:        T("Change Speed", "تغيير السرعة"),
			Preparing:     T("Preparing the video…", "جار تجهيز الفيديو…"),
			PreviewFailed: T("Couldn't show the preview", "تعذّر عرض المعاينة"),
			Chips:         chips,
		}, sel)
		w, h = vwin.speed.Size()
		pw, ph = vwin.speed.PreviewSize() // no thumbnails; the preview plays
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
	vwin.cancel, vwin.ctx = cancel, ctx
	vwin.req = make(chan float64, 1)
	vwin.mu.Lock()
	vwin.gen++
	gen := vwin.gen
	vwin.upReady, vwin.upFailed, vwin.upFrame, vwin.upThumbs, vwin.upProg = false, false, nil, map[int]*image.RGBA{}, -1
	vwin.mu.Unlock()

	refreshVideoWindow()
	popOpenLayered(vwin.wnd, vwin.last, vwin.x, vwin.y, func() { pShowWindow.Call(vwin.wnd, SW_SHOW) })
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
	thumbs, at := vwin.upThumbs, vwin.upAt
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
	if d := vwin.split; d != nil {
		d.Progress = prog
		if failed {
			d.Failed = true
		}
		if ready && !failed && d.Duration == 0 {
			vwin.info = info
			d.SetVideo(info.Duration, conv.SplitGap(info))
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
	if d := vwin.speed; d != nil {
		d.Progress = prog
		if failed {
			d.Failed = true
		}
		if ready && !failed && d.Duration == 0 {
			vwin.info = info
			d.Duration = info.Duration
			speedPlay(0)
		}
		if frame != nil {
			d.Frame = frame
			vwin.playAt = at
		}
	}
}

// speedPlay (re)starts the Speed window's silent preview at the chosen
// speed from from seconds; at the end it starts again from the beginning.
// It stops when the window closes or the speed changes.
func speedPlay(from float64) {
	d := vwin.speed
	if d == nil || vwin.ctx == nil {
		return
	}
	if vwin.play != nil {
		vwin.play()
	}
	ctx, stop := context.WithCancel(vwin.ctx)
	vwin.play = stop
	vwin.mu.Lock()
	gen := vwin.gen
	vwin.mu.Unlock()
	path, speed := vwin.path, conv.Speeds[d.Sel]
	pw, ph := d.PreviewSize()
	wnd := vwin.wnd
	logf("speed preview of %s at %s× from %.2f s", filepath.Base(path), conv.SpeedText(speed), from)
	go func() {
		for pos := from; ctx.Err() == nil; pos = 0 {
			shown := 0
			err := conv.PlayPreview(ctx, path, pos, speed, pw, ph, func(img *image.RGBA, at float64) {
				shown++
				vwin.mu.Lock()
				ok := vwin.gen == gen && ctx.Err() == nil
				if ok {
					vwin.upFrame, vwin.upAt = img, at
				}
				vwin.mu.Unlock()
				if ok {
					pPostMessageW.Call(wnd, wmVideoUpdate, 0, 0)
				}
			})
			if ctx.Err() != nil {
				return
			}
			if err != nil || shown == 0 {
				logf("speed preview stopped: %v (%d pictures)", err, shown)
				return
			}
		}
	}()
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
	case vwin.split != nil:
		img = vwin.split.Render()
	case vwin.speed != nil:
		img = vwin.speed.Render()
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
			if d := vwin.split; d != nil && d.Ready() {
				n := 1
				if (wParam == VK_LEFT) != d.RTL {
					n = -1
				}
				fr := vwin.info.FrameRate()
				splitPlayhead(d, vwin.info.FrameTime(int(math.Round(d.Playhead*fr))+n))
			}
			if d := vwin.speed; d != nil && d.Ready() {
				n := 1 // towards that side of the row (mirrored in Arabic)
				if (wParam == VK_LEFT) != d.RTL {
					n = -1
				}
				if d.Step(n) {
					speedPlay(vwin.playAt)
					refreshVideoWindow()
				}
			}
		case VK_DELETE: // removes the point last added or dragged
			if d := vwin.split; d != nil && d.Sel >= 0 {
				d.Remove(d.Sel)
				refreshVideoWindow()
			}
		}
		return 0
	case WM_LBUTTONDOWN:
		if d := vwin.speed; d != nil {
			if k, _ := d.HitTest(x, y); k == ui.SpeedNone {
				vwin.drag = true
				vwin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
			return 0
		}
		if d := vwin.split; d != nil {
			kind, i := d.HitTest(x, y)
			vwin.press = [2]int{kind, i}
			switch kind {
			case ui.SplitPoint: // drag the point; the preview follows it
				d.Sel = i
				splitPlayhead(d, d.Points[i])
				pSetCapture.Call(hwnd)
			case ui.SplitStrip: // the playhead jumps here and can be dragged
				splitPlayhead(d, d.TimeAt(x))
				pSetCapture.Call(hwnd)
			case ui.SplitNone:
				vwin.drag = true
				vwin.dragOff = POINT{int32(x), int32(y)}
				pSetCapture.Call(hwnd)
			}
			return 0
		}
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
		if d := vwin.split; d != nil {
			switch vwin.press[0] {
			case ui.SplitPoint:
				t := d.MovePoint(vwin.press[1], splitSnap(d.TimeAt(x)))
				d.Error = ""
				splitPlayhead(d, t)
				return 0
			case ui.SplitStrip:
				splitPlayhead(d, d.TimeAt(x))
				return 0
			}
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
		if vwin.split != nil && (vwin.press[0] == ui.SplitPoint || vwin.press[0] == ui.SplitStrip) {
			vwin.press = [2]int{}
		}
		return 0
	case WM_LBUTTONUP:
		if vwin.drag {
			vwin.drag = false
			pReleaseCapture.Call()
			return 0
		}
		if d := vwin.speed; d != nil {
			switch k, i := d.HitTest(x, y); k {
			case ui.SpeedClose:
				hideVideoWindow()
			case ui.SpeedChip:
				if i != d.Sel {
					d.Sel = i
					speedPlay(vwin.playAt)
					refreshVideoWindow()
				}
			case ui.SpeedButton:
				app.videoAction()
			}
			return 0
		}
		if d := vwin.split; d != nil {
			press := vwin.press
			vwin.press = [2]int{}
			if press[0] == ui.SplitPoint || press[0] == ui.SplitStrip {
				pReleaseCapture.Call()
				return 0
			}
			kind, i := d.HitTest(x, y)
			if kind != press[0] || i != press[1] {
				return 0 // pressed on one thing, released on another
			}
			switch kind {
			case ui.SplitClose:
				hideVideoWindow()
			case ui.SplitBadge:
				d.Remove(i)
				refreshVideoWindow()
			case ui.SplitAdd:
				if d.Add() {
					refreshVideoWindow()
				}
			case ui.SplitButton:
				app.videoAction()
			}
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
	if d := vwin.split; d != nil {
		a.splitStart(d)
	}
	if d := vwin.speed; d != nil {
		a.speedStart(d)
	}
}

// speedTitle is the progress window's title for a speed ("0.5×").
func speedTitle(speed float64) string {
	return fmt.Sprintf(T("Changing the speed to %s", "جار تغيير السرعة إلى %s"), conv.SpeedText(speed)+"×")
}

// speedStart closes the window and changes the speed in the progress window.
func (a *App) speedStart(d *ui.SpeedDialog) {
	if !d.Ready() {
		return
	}
	path, speed := vwin.path, conv.Speeds[d.Sel]
	cx := vwin.x + int32(float64(d.W)/2)
	cy := vwin.y + int32(float64(d.H)/2)
	scale := d.Scale
	hideVideoWindow()
	logf("speed %s: %s×", filepath.Base(path), conv.SpeedText(speed))
	a.launchVideoJobTitled(path, vwin.id, vwin.label, speedTitle(speed), cx, cy, scale, func(ctx context.Context, report func(float64)) (string, error) {
		return conv.SpeedVideo(ctx, path, speed, report)
	})
}

// splitSnap puts time t on the start of the nearest frame, so a point or
// the playhead always marks a whole frame.
func splitSnap(t float64) float64 {
	return vwin.info.FrameTime(int(math.Round(t * vwin.info.FrameRate())))
}

// splitPlayhead moves the Split window's playhead to t (on a frame start)
// and asks for that frame.
func splitPlayhead(d *ui.SplitDialog, t float64) {
	d.SetPlayhead(splitSnap(t))
	fr := vwin.info.FrameRate()
	requestFrame(vwin.info.FrameSeek(int(math.Round(d.Playhead * fr))))
	refreshVideoWindow()
}

// splitCountText is the Split window's counter line (the owner's approved
// Arabic forms by number).
func splitCountText(n int, ar bool) string {
	if !ar {
		if n == 1 {
			return "1 split point"
		}
		return fmt.Sprintf("%d split points", n)
	}
	switch {
	case n == 0:
		return "لا توجد نقاط تقسيم"
	case n == 1:
		return "نقطة تقسيم واحدة"
	case n == 2:
		return "نقطتا تقسيم"
	case n <= 10:
		return fmt.Sprintf("%d نقاط تقسيم", n)
	}
	return fmt.Sprintf("%d نقطة تقسيم", n)
}

// splitStart checks there is a point, then closes the window and starts
// the job (or shows the error line and stays open). Each point becomes the
// first frame of the next part.
func (a *App) splitStart(d *ui.SplitDialog) {
	if !d.Ready() {
		return
	}
	if len(d.Points) == 0 {
		d.Error = T("Add at least one split point", "أضف نقطة تقسيم واحدة على الأقل")
		refreshVideoWindow()
		return
	}
	fr := vwin.info.FrameRate()
	cuts := make([]float64, len(d.Points))
	for i, p := range d.Points {
		cuts[i] = vwin.info.FrameSeek(int(math.Round(p * fr)))
	}
	path := vwin.path
	cx := vwin.x + int32(float64(d.W)/2)
	cy := vwin.y + int32(float64(d.H)/2)
	scale := d.Scale
	hideVideoWindow()
	logf("split %s at %v s", filepath.Base(path), cuts)
	a.launchVideoJob(path, vwin.id, vwin.label, cx, cy, scale, func(ctx context.Context, report func(float64)) (string, error) {
		return conv.SplitVideo(ctx, path, cuts, report)
	})
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
			a.showError(T("Couldn't save the snapshot", "تعذّر حفظ اللقطة"), filepath.Base(path)+": "+conv.Cause(err))
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
