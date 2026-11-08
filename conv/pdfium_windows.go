package conv

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

var (
	pdfiumOnce sync.Once
	pdfiumErr  error
	pdfiumMu   sync.Mutex // PDFium is not thread-safe

	fpdfInit, fpdfLoadMem, fpdfClose, fpdfCount, fpdfSize, fpdfLoadPage, fpdfClosePage, fpdfRender,
	bmpCreate, bmpFill, bmpBuf, bmpStride, bmpDestroy,
	txtLoad, txtCount, txtGet, txtClose, fpdfErr,
	fpdfNewDoc, fpdfImport, fpdfSave, fpdfGetRot, fpdfSetRot *syscall.LazyProc

	saveCB  = syscall.NewCallback(pdfWriteBlock)
	saveBuf []byte // receives FPDF_SaveAsCopy output (under pdfiumMu)
)

func loadPDFium() error {
	pdfiumOnce.Do(func() {
		dll := syscall.NewLazyDLL(PDFiumPath())
		if err := dll.Load(); err != nil {
			pdfiumErr = err
			return
		}
		p := dll.NewProc
		fpdfInit, fpdfLoadMem, fpdfClose = p("FPDF_InitLibrary"), p("FPDF_LoadMemDocument64"), p("FPDF_CloseDocument")
		fpdfCount, fpdfSize = p("FPDF_GetPageCount"), p("FPDF_GetPageSizeByIndex")
		fpdfLoadPage, fpdfClosePage, fpdfRender = p("FPDF_LoadPage"), p("FPDF_ClosePage"), p("FPDF_RenderPageBitmap")
		bmpCreate, bmpFill, bmpBuf, bmpStride, bmpDestroy = p("FPDFBitmap_Create"), p("FPDFBitmap_FillRect"),
			p("FPDFBitmap_GetBuffer"), p("FPDFBitmap_GetStride"), p("FPDFBitmap_Destroy")
		txtLoad, txtCount, txtGet, txtClose = p("FPDFText_LoadPage"), p("FPDFText_CountChars"), p("FPDFText_GetText"), p("FPDFText_ClosePage")
		fpdfErr = p("FPDF_GetLastError")
		fpdfNewDoc, fpdfImport, fpdfSave = p("FPDF_CreateNewDocument"), p("FPDF_ImportPages"), p("FPDF_SaveAsCopy")
		fpdfGetRot, fpdfSetRot = p("FPDFPage_GetRotation"), p("FPDFPage_SetRotation")
		fpdfInit.Call()
	})
	return pdfiumErr
}

type winPDF struct {
	h    uintptr
	data []byte // must stay alive while the document is open
}

func openPDF(path string) (pdfDoc, error) {
	if err := loadPDFium(); err != nil {
		return nil, errors.New(tr("The PDF engine couldn't start", "تعذّر تشغيل محرك PDF"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New(tr("Empty file", "ملف فارغ"))
	}
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	h, _, _ := fpdfLoadMem.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 0)
	if h == 0 {
		code, _, _ := fpdfErr.Call()
		if code == 4 {
			return nil, errors.New(tr("This PDF is password-protected", "ملف PDF هذا محمي بكلمة مرور"))
		}
		return nil, errors.New(tr("This PDF couldn't be opened", "تعذّر فتح ملف PDF هذا"))
	}
	return &winPDF{h: h, data: data}, nil
}

func (d *winPDF) Pages() int {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	n, _, _ := fpdfCount.Call(d.h)
	return int(int32(n))
}

func (d *winPDF) Render(i int, dpi float64) (*image.RGBA, error) {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	var pw, ph float64
	if r, _, _ := fpdfSize.Call(d.h, uintptr(i), uintptr(unsafe.Pointer(&pw)), uintptr(unsafe.Pointer(&ph))); r == 0 {
		return nil, errors.New(tr("A page couldn't be read", "تعذّرت قراءة إحدى الصفحات"))
	}
	scale := dpi / 72
	// keep very large pages within a sane size
	if m := math.Max(pw, ph) * scale; m > 10000 {
		scale *= 10000 / m
	}
	w, h := int(math.Round(pw*scale)), int(math.Round(ph*scale))
	if w < 1 || h < 1 {
		return nil, errors.New(tr("A page has no size", "إحدى الصفحات بلا أبعاد"))
	}
	page, _, _ := fpdfLoadPage.Call(d.h, uintptr(i))
	if page == 0 {
		return nil, errors.New(tr("A page couldn't be read", "تعذّرت قراءة إحدى الصفحات"))
	}
	defer fpdfClosePage.Call(page)
	bmp, _, _ := bmpCreate.Call(uintptr(w), uintptr(h), 1)
	if bmp == 0 {
		return nil, errors.New(tr("Not enough memory to render the page", "لا تكفي الذاكرة لرسم الصفحة"))
	}
	defer bmpDestroy.Call(bmp)
	bmpFill.Call(bmp, 0, 0, uintptr(w), uintptr(h), 0xFFFFFFFF) // white paper
	// FPDF_ANNOT | FPDF_REVERSE_BYTE_ORDER (RGBA instead of BGRA)
	fpdfRender.Call(bmp, page, 0, 0, uintptr(w), uintptr(h), 0, 0x01|0x10)
	buf, _, _ := bmpBuf.Call(bmp)
	stride, _, _ := bmpStride.Call(bmp)
	src := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(stride)*h)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+w*4], src[y*int(stride):])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255 // paper is opaque
	}
	return img, nil
}

func (d *winPDF) Text(i int) string {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	page, _, _ := fpdfLoadPage.Call(d.h, uintptr(i))
	if page == 0 {
		return ""
	}
	defer fpdfClosePage.Call(page)
	tp, _, _ := txtLoad.Call(page)
	if tp == 0 {
		return ""
	}
	defer txtClose.Call(tp)
	n, _, _ := txtCount.Call(tp)
	if int32(n) <= 0 {
		return ""
	}
	buf := make([]uint16, int(n)+1)
	txtGet.Call(tp, 0, n, uintptr(unsafe.Pointer(&buf[0])))
	return utf16ToString(buf)
}

func (d *winPDF) Close() {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	fpdfClose.Call(d.h)
}

// fileWrite mirrors FPDF_FILEWRITE.
type fileWrite struct {
	version    int32
	writeBlock uintptr
}

func pdfWriteBlock(_ *fileWrite, data uintptr, size uintptr) uintptr {
	if size > 0 {
		saveBuf = append(saveBuf, unsafe.Slice((*byte)(unsafe.Pointer(data)), int(size))...)
	}
	return 1
}

// saveDoc returns document h as PDF bytes. Caller holds pdfiumMu.
func saveDoc(h uintptr) ([]byte, error) {
	saveBuf = nil
	fw := fileWrite{version: 1, writeBlock: saveCB}
	if r, _, _ := fpdfSave.Call(h, uintptr(unsafe.Pointer(&fw)), 0); r == 0 {
		return nil, errors.New(tr("The PDF couldn't be saved", "تعذّر حفظ ملف PDF"))
	}
	out := saveBuf
	saveBuf = nil
	return out, nil
}

func (d *winPDF) Rotated() ([]byte, error) {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	n, _, _ := fpdfCount.Call(d.h)
	for i := 0; i < int(int32(n)); i++ {
		page, _, _ := fpdfLoadPage.Call(d.h, uintptr(i))
		if page == 0 {
			return nil, errors.New(tr("A page couldn't be read", "تعذّرت قراءة إحدى الصفحات"))
		}
		r, _, _ := fpdfGetRot.Call(page)
		fpdfSetRot.Call(page, uintptr((int32(r)+1)%4))
		fpdfClosePage.Call(page)
	}
	return saveDoc(d.h)
}

func (d *winPDF) PageBytes(i int) ([]byte, error) {
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	nd, _, _ := fpdfNewDoc.Call()
	if nd == 0 {
		return nil, errors.New(tr("Not enough memory", "لا تكفي الذاكرة"))
	}
	defer fpdfClose.Call(nd)
	rng := append([]byte(fmt.Sprint(i+1)), 0)
	if r, _, _ := fpdfImport.Call(nd, d.h, uintptr(unsafe.Pointer(&rng[0])), 0); r == 0 {
		return nil, errors.New(tr("A page couldn't be copied", "تعذّر نسخ إحدى الصفحات"))
	}
	return saveDoc(nd)
}

// mergePDFs copies every page of files, in order, into one new PDF.
func mergePDFs(ctx context.Context, files []string, progress func(float64)) ([]byte, error) {
	if err := loadPDFium(); err != nil {
		return nil, errors.New(tr("The PDF engine couldn't start", "تعذّر تشغيل محرك PDF"))
	}
	var docs []*winPDF
	defer func() {
		for _, d := range docs {
			d.Close()
		}
	}()
	for _, f := range files {
		d, err := openPDF(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		docs = append(docs, d.(*winPDF))
	}
	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()
	nd, _, _ := fpdfNewDoc.Call()
	if nd == 0 {
		return nil, errors.New(tr("Not enough memory", "لا تكفي الذاكرة"))
	}
	defer fpdfClose.Call(nd)
	at := 0
	for i, d := range docs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, _, _ := fpdfCount.Call(d.h)
		if r, _, _ := fpdfImport.Call(nd, d.h, 0, uintptr(at)); r == 0 {
			return nil, fmt.Errorf(tr("%s: its pages couldn't be copied", "%s: تعذّر نسخ صفحاته"), filepath.Base(files[i]))
		}
		at += int(int32(n))
		if progress != nil {
			progress(float64(i+1) / float64(len(docs)) * 0.9)
		}
	}
	return saveDoc(nd)
}
