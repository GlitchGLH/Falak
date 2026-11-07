//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	shcore   = windows.NewLazySystemDLL("shcore.dll")
	winmm    = windows.NewLazySystemDLL("winmm.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")

	pDwmFlush   = dwmapi.NewProc("DwmFlush")
	pPlaySoundW = winmm.NewProc("PlaySoundW")

	pTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")
	pTimeEndPeriod   = winmm.NewProc("timeEndPeriod")

	pRegisterClassExW              = user32.NewProc("RegisterClassExW")
	pLoadCursorW                   = user32.NewProc("LoadCursorW")
	pSetCapture                    = user32.NewProc("SetCapture")
	pGetWindowDisplayAffinity      = user32.NewProc("GetWindowDisplayAffinity")
	pReleaseCapture                = user32.NewProc("ReleaseCapture")
	pCreateWindowExW               = user32.NewProc("CreateWindowExW")
	pDefWindowProcW                = user32.NewProc("DefWindowProcW")
	pGetMessageW                   = user32.NewProc("GetMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessageW              = user32.NewProc("DispatchMessageW")
	pPostMessageW                  = user32.NewProc("PostMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	pGetDC                         = user32.NewProc("GetDC")
	pReleaseDC                     = user32.NewProc("ReleaseDC")
	pSetTimer                      = user32.NewProc("SetTimer")
	pKillTimer                     = user32.NewProc("KillTimer")
	pSetWindowsHookExW             = user32.NewProc("SetWindowsHookExW")
	pCallNextHookEx                = user32.NewProc("CallNextHookEx")
	pUnhookWindowsHookEx           = user32.NewProc("UnhookWindowsHookEx")
	pGetCursorPos                  = user32.NewProc("GetCursorPos")
	pWindowFromPoint               = user32.NewProc("WindowFromPoint")
	pGetAncestor                   = user32.NewProc("GetAncestor")
	pGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	pGetAsyncKeyState              = user32.NewProc("GetAsyncKeyState")
	pGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	pMonitorFromPoint              = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	pAppendMenuW                   = user32.NewProc("AppendMenuW")
	pTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	pDestroyMenu                   = user32.NewProc("DestroyMenu")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pLoadImageW                    = user32.NewProc("LoadImageW")
	pMessageBoxW                   = user32.NewProc("MessageBoxW")
	pDragQueryFileW                = shell32.NewProc("DragQueryFileW")
	pShellNotifyIconW              = shell32.NewProc("Shell_NotifyIconW")
	pSHChangeNotify                = shell32.NewProc("SHChangeNotify")
	pGetDpiForMonitor              = shcore.NewProc("GetDpiForMonitor")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pBitBlt             = gdi32.NewProc("BitBlt")

	pGetModuleHandleW             = kernel32.NewProc("GetModuleHandleW")
	pGetTickCount                 = kernel32.NewProc("GetTickCount")
	pCloseHandle                  = kernel32.NewProc("CloseHandle")
	pIsWindow                     = user32.NewProc("IsWindow")
	pFindWindowW                  = user32.NewProc("FindWindowW")
	pRegisterWindowMessageW       = user32.NewProc("RegisterWindowMessageW")
	pGetLastInputInfo             = user32.NewProc("GetLastInputInfo")
	pSendInput                    = user32.NewProc("SendInput")
	pShellExecuteW                = shell32.NewProc("ShellExecuteW")
	pSHQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")
	pGetForegroundWindow          = user32.NewProc("GetForegroundWindow")
	pGetGUIThreadInfo             = user32.NewProc("GetGUIThreadInfo")
	pGetClassNameW                = user32.NewProc("GetClassNameW")
	pGetWindowRect                = user32.NewProc("GetWindowRect")
	pMonitorFromWindow            = user32.NewProc("MonitorFromWindow")
	pGetWindowLongPtrW            = user32.NewProc("GetWindowLongPtrW")
	pSetWindowLongPtrW            = user32.NewProc("SetWindowLongPtrW")
	pCreateMutexW                 = kernel32.NewProc("CreateMutexW")

	pOleInitialize    = ole32.NewProc("OleInitialize")
	pRegisterDragDrop = ole32.NewProc("RegisterDragDrop")
	pReleaseStgMedium = ole32.NewProc("ReleaseStgMedium")
)

const (
	WS_POPUP          = 0x80000000
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_NOACTIVATE  = 0x08000000
	SW_HIDE           = 0
	SW_SHOWNOACTIVATE = 4
	ULW_ALPHA         = 2
	AC_SRC_ALPHA      = 1

	WM_DESTROY       = 0x0002
	WM_TIMER         = 0x0113
	WM_COMMAND       = 0x0111
	WM_LBUTTONUP     = 0x0202
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDOWN   = 0x0201
	WM_MOUSEMOVE     = 0x0200
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105
	WM_NULL          = 0x0000
	WM_APP           = 0x8000
	WM_MOUSEACTIVATE = 0x0021
	MA_NOACTIVATE    = 3

	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14

	VK_SHIFT   = 0x10
	VK_MENU    = 0x12
	VK_CONTROL = 0x11
	VK_ESCAPE  = 0x1B
	VK_TAB     = 0x09
	VK_LSHIFT  = 0xA0
	VK_RSHIFT  = 0xA1
	VK_LMENU   = 0xA4
	VK_RMENU   = 0xA5
	VK_LBUTTON = 0x01

	SM_CXDRAG = 68
	SM_CYDRAG = 69

	DROPEFFECT_NONE = 0
	DROPEFFECT_COPY = 1

	CF_HDROP         = 15
	DVASPECT_CONTENT = 1
	TYMED_HGLOBAL    = 1

	MF_STRING        = 0x0000
	MF_GRAYED        = 0x0001
	MF_CHECKED       = 0x0008
	MF_SEPARATOR     = 0x0800
	MFT_RIGHTORDER   = 0x2000
	MFT_RIGHTJUSTIFY = 0x4000
	TPM_RETURNCMD    = 0x0100
	TPM_RIGHTBUTTON  = 0x0002
	TPM_RIGHTALIGN   = 0x0008
	TPM_BOTTOMALIGN  = 0x0020
	TPM_LAYOUTRTL    = 0x8000

	NIM_ADD     = 0
	NIM_MODIFY  = 1
	NIM_DELETE  = 2
	NIF_MESSAGE = 0x1
	NIF_ICON    = 0x2
	NIF_TIP     = 0x4
	NIF_INFO    = 0x10
	NIIF_INFO   = 0x1
	NIIF_ERROR  = 0x3

	MB_OK         = 0
	MB_ICONINFO   = 0x40
	MB_ICONERROR  = 0x10
	MB_RIGHT      = 0x80000
	MB_RTLREADING = 0x100000
	MB_TOPMOST    = 0x40000

	SRCCOPY                  = 0x00CC0020
	CAPTUREBLT               = 0x40000000
	BI_RGB                   = 0
	GA_ROOT                  = 2
	MONITOR_DEFAULTTONEAREST = 2
	MDT_EFFECTIVE_DPI        = 0
	IMAGE_ICON               = 1

	SHCNE_CREATE   = 0x00000002
	SHCNF_PATHW    = 0x0005
	SWP_NOACTIVATE = 0x0010
	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	HWND_TOPMOST   = ^uintptr(0) // (HWND)-1
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	_       uint32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSLLHOOKSTRUCT struct {
	Pt          POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BLENDFUNCTION struct {
	BlendOp, BlendFlags, SourceConstantAlpha, AlphaFormat byte
}

type SIZE struct{ CX, CY int32 }

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         GUID
	HBalloonIcon     uintptr
}

type FORMATETC struct {
	CfFormat uint16
	Ptd      uintptr
	DwAspect uint32
	Lindex   int32
	Tymed    uint32
}

type STGMEDIUM struct {
	Tymed          uint32
	HGlobal        uintptr
	PUnkForRelease uintptr
}

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func copyUTF16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}

func loword(v uintptr) uint32 { return uint32(v & 0xFFFF) }

func ptr[T any](p *T) uintptr { return uintptr(unsafe.Pointer(p)) }

type MOUSEINPUT struct {
	Dx, Dy      int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	Mi   MOUSEINPUT
}

type GUITHREADINFO struct {
	CbSize        uint32
	Flags         uint32
	HwndActive    uintptr
	HwndFocus     uintptr
	HwndCapture   uintptr
	HwndMenuOwner uintptr
	HwndMoveSize  uintptr
	HwndCaret     uintptr
	RcCaret       RECT
}

var pSetFileAttributesW = kernel32.NewProc("SetFileAttributesW")

// hidePath gives a path the hidden and system attributes, so Explorer does
// not show it even when "show hidden files" is on.
func hidePath(p string) {
	pSetFileAttributesW.Call(uintptr(unsafe.Pointer(utf16(p))), 0x2|0x4)
}
