package conv

// Windows Imaging Component: decodes any image format Windows itself can
// open (including camera RAW with the Raw Image Extension).

import (
	"errors"
	"image"
	"runtime"
	"syscall"
	"unsafe"
)

type guid struct {
	d1     uint32
	d2, d3 uint16
	d4     [8]byte
}

var (
	ole32w             = syscall.NewLazyDLL("ole32.dll")
	pCoInitializeEx    = ole32w.NewProc("CoInitializeEx")
	pCoUninitialize    = ole32w.NewProc("CoUninitialize")
	pCoCreateInstanceW = ole32w.NewProc("CoCreateInstance")

	clsidWICFactory = guid{0xcacaf262, 0x9370, 0x4615, [8]byte{0xa1, 0x3b, 0x9f, 0x55, 0x39, 0xda, 0x4c, 0x0a}}
	iidWICFactory   = guid{0xec5ec8a9, 0xc395, 0x4314, [8]byte{0x9c, 0x77, 0x54, 0xd7, 0xa9, 0x35, 0xff, 0x70}}
	wicFormatRGBA   = guid{0xf5c7ad2d, 0x6a8d, 0x43dd, [8]byte{0xa7, 0xa8, 0xa2, 0x99, 0x35, 0x26, 0x1a, 0xe9}}
)

func init() { decodeRAWNative = wicDecode }

// call invokes method #idx of a COM object.
func call(obj uintptr, idx int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func release(obj uintptr) {
	if obj != 0 {
		call(obj, 2)
	}
}

func wicDecode(path string) (image.Image, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if hr, _, _ := pCoInitializeEx.Call(0, 0); int32(hr) >= 0 { // COINIT_MULTITHREADED
		defer pCoUninitialize.Call()
	}
	var factory uintptr
	if hr, _, _ := pCoCreateInstanceW.Call(uintptr(unsafe.Pointer(&clsidWICFactory)), 0, 1,
		uintptr(unsafe.Pointer(&iidWICFactory)), uintptr(unsafe.Pointer(&factory))); int32(hr) < 0 {
		return nil, errors.New("WIC unavailable")
	}
	defer release(factory)
	p, _ := syscall.UTF16PtrFromString(path)
	var decoder uintptr
	// IWICImagingFactory::CreateDecoderFromFilename
	if hr := call(factory, 3, uintptr(unsafe.Pointer(p)), 0, 0x80000000, 0, uintptr(unsafe.Pointer(&decoder))); int32(hr) < 0 {
		return nil, errors.New("no WIC decoder for this file")
	}
	defer release(decoder)
	var frame uintptr
	if hr := call(decoder, 13, 0, uintptr(unsafe.Pointer(&frame))); int32(hr) < 0 { // GetFrame(0)
		return nil, errors.New("WIC GetFrame failed")
	}
	defer release(frame)
	var conv uintptr
	if hr := call(factory, 10, uintptr(unsafe.Pointer(&conv))); int32(hr) < 0 { // CreateFormatConverter
		return nil, errors.New("WIC converter failed")
	}
	defer release(conv)
	// IWICFormatConverter::Initialize(source, format, dither none, palette nil, alpha 0.0, custom)
	if hr := call(conv, 8, frame, uintptr(unsafe.Pointer(&wicFormatRGBA)), 0, 0, 0, 0); int32(hr) < 0 {
		return nil, errors.New("WIC format conversion failed")
	}
	var w, h uint32
	call(conv, 3, uintptr(unsafe.Pointer(&w)), uintptr(unsafe.Pointer(&h))) // GetSize
	if w == 0 || h == 0 || uint64(w)*uint64(h) > 400_000_000 {
		return nil, errors.New("bad size")
	}
	img := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
	stride := uint32(img.Stride)
	if hr := call(conv, 7, 0, uintptr(stride), uintptr(len(img.Pix)), uintptr(unsafe.Pointer(&img.Pix[0]))); int32(hr) < 0 { // CopyPixels
		return nil, errors.New("WIC CopyPixels failed")
	}
	return img, nil
}

// WICDecodeFile exposes the WIC decoder (used by tests and diagnostics).
func WICDecodeFile(path string) (image.Image, error) { return wicDecode(path) }
