package main

import (
	"golang.org/x/sys/windows/registry"

	"raha/conv"
)

// English is the default language; Arabic can be chosen from the tray menu.
var arabic bool

func T(en, ar string) string {
	if arabic {
		return ar
	}
	return en
}

func loadLanguage() {
	arabic = false
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Falak`, registry.QUERY_VALUE); err == nil {
		v, _, err := k.GetStringValue("Language")
		k.Close()
		arabic = err == nil && v == "ar"
	}
	applyLanguage()
}

func setLanguage(ar bool) {
	if k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS); err == nil {
		if ar {
			k.SetStringValue("Language", "ar")
		} else {
			k.DeleteValue("Language")
		}
		k.Close()
	}
	arabic = ar
	applyLanguage()
}

func applyLanguage() {
	conv.Arabic = arabic
	appName = T("Falak", "فلك")
	appVersion = appVersionNum
}

// rtl gives message-box flags for right-to-left text in Arabic.
func rtl() uintptr {
	if arabic {
		return MB_RIGHT | MB_RTLREADING
	}
	return 0
}

func idleTip() string {
	return T("Falak — drag a file and press Shift", "فلك — اسحب ملفًا واضغط Shift")
}

func helpText() string {
	return T(`How to use Falak:

1. Start dragging a file (or several) from File Explorer, the desktop or any app that lets you drag files.
2. While dragging, press Shift: the wheel appears around the pointer.
3. Drop the file on the format you want. A new file is saved next to the original; the original is never changed.

• Press Alt, Ctrl or Tab while the wheel is showing to switch between formats and tools.
• Dropping in the middle of the wheel or outside it, or pressing Esc, cancels.
• A small window shows the progress; drag it anywhere, or press its X to stop.
• Select two or more images (or PDFs) and use Merge in the tools to make one PDF.
• Everything happens on your computer. Some formats need a one-time download of a free engine (FFmpeg for video and audio, PDFium and qpdf for PDF, resvg for SVG).

Supported: images (JPG, PNG, WEBP, AVIF, HEIC, TIFF, BMP, GIF, ICO, SVG, camera RAW), PDF, Word / Excel / PowerPoint (with Microsoft Office or LibreOffice), video (MP4, MOV, MKV, WEBM, AVI, WMV, GIF), audio (MP3, M4A, WAV, FLAC, OGG, OPUS, WMA, AIFF), subtitles (SRT, VTT, ASS, SSA), text files (TXT to PDF, DOCX, RTF, HTML, JPG, PNG, SRT, VTT) and archives (ZIP, 7Z, RAR, TAR, GZIP).

Engines downloaded on first use belong to their authors and keep their own licenses: FFmpeg (GPL), PDFium (BSD), qpdf (Apache 2.0), resvg (Apache 2.0 / MIT).

— Falak By UncleGlitch`,
		`طريقة الاستخدام:

١. ابدأ سحب ملف (أو عدة ملفات) من مستكشف الملفات أو سطح المكتب أو أي برنامج يسمح بسحب الملفات.
٢. أثناء السحب اضغط Shift، فتظهر العجلة حول المؤشر.
٣. أفلت الملف على الصيغة التي تريدها، فتُحفظ نسخة جديدة بجانب الملف الأصلي، ولا يتغير الأصل.

• اضغط Alt أو Ctrl أو Tab والعجلة ظاهرة للتبديل بين الصيغ والأدوات.
• الإفلات في وسط العجلة أو خارجها، أو الضغط على Esc، يلغي العملية.
• تظهر نافذة صغيرة بنسبة التقدّم، يمكنك سحبها إلى أي مكان، أو الضغط على X فيها لإيقاف العملية.
• حدّد صورتين أو أكثر (أو ملفي PDF أو أكثر) واستخدم Merge في الأدوات لدمجها في ملف PDF واحد.
• كل شيء يتم على جهازك. بعض الصيغ تحتاج تنزيل محرك مجاني مرة واحدة فقط (FFmpeg للفيديو والصوت، وPDFium وqpdf لملفات PDF، وresvg لملفات SVG).

المدعوم: الصور (JPG وPNG وWEBP وAVIF وHEIC وTIFF وBMP وGIF وICO وSVG وصور الكاميرا RAW)، وPDF، وملفات Word وExcel وPowerPoint (مع Microsoft Office أو LibreOffice)، والفيديو (MP4 وMOV وMKV وWEBM وAVI وWMV وGIF)، والصوت (MP3 وM4A وWAV وFLAC وOGG وOPUS وWMA وAIFF)، والترجمات (SRT وVTT وASS وSSA)، والملفات النصية (TXT إلى PDF وDOCX وRTF وHTML وJPG وPNG وSRT وVTT)، والأرشيفات (ZIP و7Z وRAR وTAR وGZIP).

المحركات التي تُنزَّل عند أول استخدام ملك لأصحابها وتخضع لتراخيصها: FFmpeg ‏(GPL)، وPDFium ‏(BSD)، وqpdf ‏(Apache 2.0)، وresvg ‏(Apache 2.0 / MIT).

— فلك، By UncleGlitch`)
}
