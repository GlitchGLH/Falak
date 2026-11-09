//go:build windows

package main

import "testing"

func TestBlockDecision(t *testing.T) {
	mon := RECT{0, 0, 1366, 768}
	full := RECT{0, 0, 1366, 768}
	const popup = WS_POPUP | 0x10000000 // WS_POPUP | WS_VISIBLE
	explorer := &winInfo{Class: "CabinetWClass", Process: "explorer.exe", Rect: full, Style: popup}
	overlay := &winInfo{Class: "CEF-OSC-WIDGET", Process: "NVIDIA Overlay.exe", Rect: full, Style: popup,
		ExStyle: WS_EX_TOPMOST | WS_EX_LAYERED | WS_EX_TRANSPARENT}
	layeredTop := &winInfo{Class: "Chrome_WidgetWin_1", Process: "Avira.Spotlight.UI.Application.exe", Rect: full, Style: popup,
		ExStyle: WS_EX_TOPMOST | WS_EX_LAYERED}
	own := &winInfo{Class: "FalakWheel", Process: "Falak.exe", Rect: full, Style: popup, Own: true}
	game := &winInfo{Class: "UnityWndClass", Process: "StarRail.exe", Rect: full, Style: popup}
	video := &winInfo{Class: "Qt5QWindowIcon", Process: "vlc.exe", Rect: RECT{-1, -1, 1367, 769}, Style: popup}
	normal := &winInfo{Class: "Notepad", Process: "notepad.exe", Rect: RECT{100, 100, 900, 600}, Style: WS_THICKFRAME}
	maximized := &winInfo{Class: "Notepad", Process: "notepad.exe", Rect: RECT{-8, -8, 1374, 776}, Style: WS_THICKFRAME}
	progman := &winInfo{Class: "Progman", Rect: full, Style: popup}
	workerw := &winInfo{Class: "WorkerW", Rect: full, Style: popup}

	for _, c := range []struct {
		name  string
		state int32
		fg    *winInfo
		want  string
	}{
		{"QUNS_BUSY + Explorer", 2, explorer, ""},
		{"QUNS_BUSY + transparent topmost overlay", 2, overlay, ""},
		{"QUNS_BUSY + layered topmost overlay", 2, layeredTop, ""},
		{"QUNS_BUSY + Falak window", 2, own, ""},
		{"QUNS_BUSY + desktop Progman", 2, progman, ""},
		{"QUNS_BUSY + desktop WorkerW", 2, workerw, ""},
		{"QUNS_BUSY + no foreground window", 2, nil, ""},
		{"QUNS_BUSY + normal window", 2, normal, ""},
		{"QUNS_BUSY + maximized window", 2, maximized, ""},
		{"QUNS_BUSY + borderless full-screen game", 2, game, "a full-screen app is running"},
		{"QUNS_BUSY + full-screen video player", 2, video, "a full-screen app is running"},
		{"state 3 (D3D full screen)", 3, explorer, "a full-screen game is running"},
		{"state 3, no foreground window", 3, nil, "a full-screen game is running"},
		{"state 4 (presentation)", 4, explorer, "presentation mode"},
		{"state 0 + normal window", 0, normal, ""},
		{"state 5 + normal window", 5, normal, ""},
		{"state 5 + Explorer", 5, explorer, ""},
		{"state 0 + full-screen video player", 0, video, "the active window is full screen"},
		{"state 5 + borderless full-screen game", 5, game, "the active window is full screen"},
	} {
		if got := blockDecision(c.state, c.fg, mon); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
