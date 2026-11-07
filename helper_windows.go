//go:build windows

package main

import (
	"os"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// On some systems Windows hides the keyboard and mouse-button state from a
// normal program while a drag is in progress, but still shows it to a program
// running as administrator. However, a program running as administrator
// cannot receive files dropped from File Explorer. So Falak is split in two:
//
//   - the main process runs normally (drops work), and
//   - a tiny helper started with "--keys <hwnd>" runs as administrator and
//     only reads the Shift/Alt/Ctrl/Tab/Esc keys and the mouse button,
//     posting the state to the main window. Messages from a higher level to
//     a lower level are allowed by Windows.

const (
	wmKeys   = WM_APP + 20
	keyLB    = 1
	keyShift = 2
	keyTog   = 4
	keyEsc   = 8
)

func findMainWindow() uintptr {
	h, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(utf16("FalakMsg"))), 0)
	return h
}

func runKeysHelper(arg string) {
	if v, err := strconv.ParseUint(arg, 0, 64); err == nil && v != 0 {
		_ = v // older argument form; the window is now found by class name
	}
	logf("helper started (level %s)", ilName(selfIntegrity()))
	if selfIntegrity() >= ilHigh {
		registerHelperTask()
	}
	parent := findMainWindow()
	lost := time.Now()
	last := uint32(0xFFFF)
	beat := time.Now()
	checked := time.Now()
	for {
		if time.Since(checked) > time.Second {
			checked = time.Now()
			if !useHelperSetting() {
				logf("helper: turned off from the menu, exiting")
				return
			}
		}
		if ok, _, _ := pIsWindow.Call(parent); ok == 0 {
			parent = findMainWindow()
			if parent == 0 {
				if time.Since(lost) > 5*time.Second {
					logf("helper: Falak is not running, exiting")
					return
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}
		}
		lost = time.Now()
		var m uint32
		if leftButtonDown() {
			m |= keyLB
		}
		if keyDown(VK_SHIFT) {
			m |= keyShift
		}
		if keyDown(VK_MENU) || keyDown(VK_CONTROL) || keyDown(VK_TAB) {
			m |= keyTog
		}
		if keyDown(VK_ESCAPE) {
			m |= keyEsc
		}
		if m != last || time.Since(beat) > 400*time.Millisecond {
			pPostMessageW.Call(parent, wmKeys, uintptr(m), 1)
			last, beat = m, time.Now()
		}
		time.Sleep(8 * time.Millisecond)
	}
}

var (
	helperMask  uint32
	helperSeen  time.Time
	helperAsked bool
)

func helperAlive() bool { return time.Since(helperSeen) < time.Second }

func useHelperSetting() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Falak`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("UseHelper")
	return err == nil && v == 1
}

func setUseHelper(on bool) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS)
	if err != nil {
		return
	}
	defer k.Close()
	if on {
		k.SetDWordValue("UseHelper", 1)
	} else {
		k.DeleteValue("UseHelper")
	}
}

var helperLaunchedAt time.Time

func (a *App) launchHelper() bool {
	helperLaunchedAt = time.Now()
	if startHelperTask() {
		return true
	}
	exe := installedExe()
	if _, err := os.Stat(exe); err != nil {
		exe, _ = os.Executable()
	}
	r, _, _ := pShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16("runas"))), uintptr(unsafe.Pointer(utf16(exe))),
		uintptr(unsafe.Pointer(utf16("--keys"))), 0, 0)
	logf("launching helper as administrator from %s: result %d", exe, r)
	return r > 32
}

// offerHelper runs (outside the poll loop) when a drag is seen but the mouse
// button reads as not pressed, i.e. Windows is hiding input from Falak.
func (a *App) offerHelper() {
	if helperAlive() {
		return
	}
	if useHelperSetting() {
		// Enabled but not running (e.g. it failed to start): retry quietly.
		if time.Since(helperLaunchedAt) > 20*time.Second {
			a.launchHelper()
		}
		return
	}
	if helperAsked {
		return
	}
	helperAsked = true
	msg := T("When you drag from the desktop, Windows hides the Shift key and mouse button from Falak, so the wheel can't appear there.\n\n"+
		"The fix is a small helper with administrator rights whose only job is to read the keys and pass them to Falak. "+
		"Windows will ask for your approval once.\n\nStart the helper now?",
		"عند السحب من سطح المكتب يُخفي ويندوز عن «فلك» حالة زر Shift وزر الفأرة، ولهذا لا تظهر العجلة هناك.\n\n"+
			"الحل تشغيل مساعد صغير بصلاحيات المسؤول، وظيفته الوحيدة قراءة حالة الأزرار وإرسالها إلى «فلك». "+
			"سيطلب ويندوز موافقتك مرة واحدة فقط.\n\nهل تريد تشغيل المساعد الآن؟")
	r, _, _ := pMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16(msg))), uintptr(unsafe.Pointer(utf16(appName))),
		0x4|0x20|rtl()|MB_TOPMOST)
	if r == 6 {
		setUseHelper(true)
		a.launchHelper()
	}
}

// relaunchNormal restarts Falak at the normal level when it was started as
// administrator, because then File Explorer cannot drop files on the wheel.
func relaunchNormal() bool {
	// Guard against a loop on systems where every program runs as
	// administrator (UAC turned off): relaunch at most once per 15 seconds.
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS)
	if err == nil {
		last, _, _ := k.GetIntegerValue("RelaunchedAt")
		now := uint64(time.Now().Unix())
		if now-last < 15 {
			k.Close()
			logf("still running as administrator after relaunch (UAC may be off); continuing")
			return false
		}
		k.SetQWordValue("RelaunchedAt", now)
		k.Close()
	}
	exe, _ := os.Executable()
	logf("started as administrator; restarting at normal level via File Explorer")
	if mutexHandle != 0 {
		pCloseHandle.Call(mutexHandle)
		mutexHandle = 0
	}
	r, _, _ := pShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16("explorer.exe"))),
		uintptr(unsafe.Pointer(utf16(`"`+exe+`"`))), 0, 1)
	return r > 32
}
