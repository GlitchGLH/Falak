//go:build windows

package main

import "strings"

// The full-screen rule, free of Windows calls so it can be tested.

const (
	qunsBusy         = 2 // QUNS_BUSY: Windows thinks some full-screen window is up
	qunsD3DFull      = 3 // QUNS_RUNNING_D3D_FULL_SCREEN
	qunsPresentation = 4 // QUNS_PRESENTATION_MODE
)

// countsAsFullScreen: w is a real full-screen window (a game, a video): it
// has no resizable frame and covers the monitor, and it is none of Falak's
// own windows, not Explorer or the desktop, and not a click-through overlay
// (NVIDIA Overlay and similar are transparent, or layered and topmost).
func countsAsFullScreen(w winInfo, mon RECT) bool {
	switch {
	case w.Own,
		strings.EqualFold(w.Process, "explorer.exe"),
		w.Class == "Progman", w.Class == "WorkerW",
		w.ExStyle&WS_EX_TRANSPARENT != 0,
		w.ExStyle&WS_EX_LAYERED != 0 && w.ExStyle&WS_EX_TOPMOST != 0:
		return false
	}
	return w.Style&WS_THICKFRAME == 0 && rectCovers(w.Rect, mon)
}

// blockDecision returns why the wheel must not open, or "". state is
// SHQueryUserNotificationState's value (0 when the call failed), fg the
// foreground window (nil when there is none) and mon its monitor's rect.
// QUNS_BUSY alone does not block: an overlay can set it while the user is
// only in File Explorer.
func blockDecision(state int32, fg *winInfo, mon RECT) string {
	switch state {
	case qunsD3DFull:
		return "a full-screen game is running"
	case qunsPresentation:
		return "presentation mode"
	}
	if fg == nil || !countsAsFullScreen(*fg, mon) {
		return ""
	}
	if state == qunsBusy {
		return "a full-screen app is running"
	}
	return "the active window is full screen"
}
