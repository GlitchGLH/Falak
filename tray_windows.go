//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func (a *App) trayData() NOTIFYICONDATAW {
	nid := NOTIFYICONDATAW{HWnd: a.msgWnd, UID: 1}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	return nid
}

func (a *App) addTray() {
	nid := a.trayData()
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.UCallbackMessage = wmTray
	smX, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
	smY, _, _ := pGetSystemMetrics.Call(50)
	nid.HIcon, _, _ = pLoadImageW.Call(a.hInst, 1, IMAGE_ICON, smX, smY, 0)
	copyUTF16(nid.SzTip[:], idleTip())
	pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) removeTray() {
	nid := a.trayData()
	pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) balloon(title, text string, isErr bool) {
	nid := a.trayData()
	nid.UFlags = NIF_INFO
	copyUTF16(nid.SzInfoTitle[:], title)
	copyUTF16(nid.SzInfo[:], text)
	nid.DwInfoFlags = NIIF_INFO
	if isErr {
		nid.DwInfoFlags = NIIF_ERROR
	}
	r, _, e := pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	// Shell_NotifyIcon accepts the notification even when Windows will not
	// show it (notifications or the notification center turned off, Do not
	// disturb), so the log also says what the settings read.
	if r == 0 {
		logf("tray notification %q: Shell_NotifyIcon failed (%v); %s", title, e, notificationState())
	} else {
		logf("tray notification %q: Shell_NotifyIcon ok; %s", title, notificationState())
	}
}

// notificationState reads (never changes) the settings that silence tray
// notifications, for the log.
func notificationState() string {
	read := func(root registry.Key, path, name string) string {
		k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
		if err != nil {
			return "-"
		}
		defer k.Close()
		v, _, err := k.GetIntegerValue(name)
		if err != nil {
			return "-"
		}
		return fmt.Sprint(v)
	}
	return fmt.Sprintf("ToastEnabled=%s DisableNotificationCenter(HKCU)=%s (HKLM)=%s",
		read(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\PushNotifications`, "ToastEnabled"),
		read(registry.CURRENT_USER, `Software\Policies\Microsoft\Windows\Explorer`, "DisableNotificationCenter"),
		read(registry.LOCAL_MACHINE, `Software\Policies\Microsoft\Windows\Explorer`, "DisableNotificationCenter"))
}

func (a *App) setTrayTip(text string) {
	nid := a.trayData()
	nid.UFlags = NIF_TIP
	copyUTF16(nid.SzTip[:], text)
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}
