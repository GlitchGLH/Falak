//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// A minimal hand-built COM IDropTarget. The wheel window registers it so that
// Explorer hands us the dragged files. We only ever answer DROPEFFECT_COPY or
// DROPEFFECT_NONE, so Explorer never deletes or moves the original file even
// though Shift normally means "move" during a drag.

type dropTargetVtbl struct {
	QueryInterface, AddRef, Release, DragEnter, DragOver, DragLeave, Drop uintptr
}

type dropTarget struct {
	vtbl *dropTargetVtbl
	refs int32
}

var (
	iidIUnknown    = GUID{0x00000000, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIDropTarget = GUID{0x00000122, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

	theVtbl       *dropTargetVtbl
	theDropTarget = &dropTarget{refs: 1}
)

func init() {
	theVtbl = &dropTargetVtbl{
		QueryInterface: syscall.NewCallback(dtQueryInterface),
		AddRef:         syscall.NewCallback(dtAddRef),
		Release:        syscall.NewCallback(dtRelease),
		DragEnter:      syscall.NewCallback(dtDragEnter),
		DragOver:       syscall.NewCallback(dtDragOver),
		DragLeave:      syscall.NewCallback(dtDragLeave),
		Drop:           syscall.NewCallback(dtDrop),
	}
	theDropTarget.vtbl = theVtbl
}

const (
	sOK          = 0
	eNoInterface = 0x80004002
)

func dtQueryInterface(this, riid, ppv uintptr) uintptr {
	id := (*GUID)(unsafe.Pointer(riid))
	out := (*uintptr)(unsafe.Pointer(ppv))
	if *id == iidIUnknown || *id == iidIDropTarget {
		*out = this
		theDropTarget.refs++
		return sOK
	}
	*out = 0
	return eNoInterface
}

func dtAddRef(this uintptr) uintptr  { theDropTarget.refs++; return uintptr(theDropTarget.refs) }
func dtRelease(this uintptr) uintptr { theDropTarget.refs--; return uintptr(theDropTarget.refs) }

func ptFromPointl(v uintptr) POINT { return POINT{int32(uint32(v)), int32(uint32(v >> 32))} }

func setEffect(p uintptr, e uint32) {
	if p != 0 {
		*(*uint32)(unsafe.Pointer(p)) = e
	}
}

func dtDragEnter(this, dataObj, keyState, pt, effect uintptr) uintptr {
	paths := readDropPaths(dataObj)
	setEffect(effect, app.onDragEnter(paths, ptFromPointl(pt)))
	return sOK
}

func dtDragOver(this, keyState, pt, effect uintptr) uintptr {
	setEffect(effect, app.onDragOver(ptFromPointl(pt)))
	return sOK
}

func dtDragLeave(this uintptr) uintptr {
	app.onDragLeave()
	return sOK
}

func dtDrop(this, dataObj, keyState, pt, effect uintptr) uintptr {
	paths := readDropPaths(dataObj)
	setEffect(effect, app.onDrop(paths, ptFromPointl(pt)))
	return sOK
}

// readDropPaths extracts CF_HDROP file paths from an IDataObject.
func readDropPaths(obj uintptr) []string {
	if obj == 0 {
		return nil
	}
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	getData := *(*uintptr)(unsafe.Pointer(vtbl + 3*unsafe.Sizeof(uintptr(0))))
	fe := FORMATETC{CfFormat: CF_HDROP, DwAspect: DVASPECT_CONTENT, Lindex: -1, Tymed: TYMED_HGLOBAL}
	var med STGMEDIUM
	hr, _, _ := syscall.SyscallN(getData, obj, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&med)))
	if hr != sOK || med.HGlobal == 0 {
		return nil
	}
	defer pReleaseStgMedium.Call(uintptr(unsafe.Pointer(&med)))
	n, _, _ := pDragQueryFileW.Call(med.HGlobal, 0xFFFFFFFF, 0, 0)
	var paths []string
	for i := uintptr(0); i < n; i++ {
		l, _, _ := pDragQueryFileW.Call(med.HGlobal, i, 0, 0)
		buf := make([]uint16, l+1)
		pDragQueryFileW.Call(med.HGlobal, i, uintptr(unsafe.Pointer(&buf[0])), l+1)
		paths = append(paths, syscall.UTF16ToString(buf))
	}
	return paths
}
