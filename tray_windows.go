//go:build windows

package main

import (
	"unsafe"
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
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) setTrayTip(text string) {
	nid := a.trayData()
	nid.UFlags = NIF_TIP
	copyUTF16(nid.SzTip[:], text)
	pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}
