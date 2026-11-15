# Falak

Designed and directed by UncleGlitch, built with AI assistance.

**Falak** is a file conversion wheel. Drag a file (or several), press **Shift**
while dragging, and drop it on the format you want — a new file is saved next
to the original, which is never changed.

By UncleGlitch (GitHub: @GlitchGLH)

[Download the latest release](https://github.com/GlitchGLH/Falak/releases/latest)

<p>
  <img src="docs/wheel-light.png" width="49%" alt="The wheel, light theme">
  <img src="docs/wheel-dark.png" width="49%" alt="The wheel, dark theme">
</p>

<p>
  <img src="docs/tools-wheel-light.png" width="49%" alt="The tools wheel, light theme">
  <img src="docs/tools-wheel-dark.png" width="49%" alt="The tools wheel, dark theme">
</p>

<p>
  <img src="docs/tray-menu-en-light.png" width="49%" alt="The tray menu in English, light theme">
  <img src="docs/tray-menu-en-dark.png" width="49%" alt="The tray menu in English, dark theme">
</p>

<p>
  <img src="docs/compress-dialog-en-light.png" width="49%" alt="The Compress dialog in English, light theme">
  <img src="docs/compress-dialog-en-dark.png" width="49%" alt="The Compress dialog in English, dark theme">
</p>

<p>
  <img src="docs/progress-en-light.png" width="49%" alt="The progress window in English, light theme">
  <img src="docs/progress-en-dark.png" width="49%" alt="The progress window in English, dark theme">
</p>

<p>
  <img src="docs/how-to-use-en-light.png" width="49%" alt="The How to use card in English, light theme">
  <img src="docs/how-to-use-en-dark.png" width="49%" alt="The How to use card in English, dark theme">
</p>

## Supported formats


| Category | Formats |
|---|---|
| Images | JPG, PNG, WEBP, AVIF, HEIC, TIFF, BMP, GIF, ICO, SVG, camera RAW |
| PDF | PDF |
| Word / Excel / PowerPoint | with Microsoft Office or LibreOffice |
| Video | MP4, MOV, MKV, WEBM, AVI, WMV, GIF |
| Audio | MP3, M4A, WAV, FLAC, OGG, OPUS, WMA, AIFF |
| Subtitles | SRT, VTT, ASS, SSA |
| Text files | TXT to PDF, DOCX, RTF, HTML, JPG, PNG, SRT, VTT |
| Archives | ZIP, 7Z, RAR, TAR, GZIP; a ZIP, RAR or 7Z that holds only images converts to PDF |

An archive of images becomes one PDF with one image per page, in natural
order (2 before 10), next to the archive. Files such as `Thumbs.db` and
`desktop.ini` inside it are ignored.

## Tools

Press **Alt**, **Ctrl** or **Tab** while the wheel is showing to switch
between formats and tools. A tool that doesn't fit the dragged files is shown
disabled, never hidden.

- **Video:** COMPRESS, TRIM, METADATA, MUTE, SPEED, SNAPSHOT, SPLIT, CROP
- **Images:** COMPRESS, METADATA, ROTATE, RESIZE 50%, WHITE BG, BLACK BG
  (transparent images), MERGE PDF (two or more images)
- **Audio:** COMPRESS, METADATA, NORMALIZE, VISUALIZE, CHANNELS
- **PDF:** COMPRESS, ROTATE, SPLIT, MERGE (two or more PDFs)
- **Archives:** EXTRACT

**Compressing several files** at once saves the copies in a new folder named
after the date and time, such as `Compressed 2026-11-15 05-30`, next to the
originals. Each copy keeps its original name. One file is saved next to the
original as before.

**Messages and questions** appear in Falak's own window, in its style and
language, instead of Windows message boxes. Press Ctrl+C to copy a message.

## Video windows

TRIM, SNAPSHOT, SPLIT, SPEED and CROP open a window with a preview of the
video. They work on one video at a time (with several videos dragged they are
disabled). The preview is silent; the saved files keep their sound. Press
Enter to run, or Esc to close.

- **Trim Video:** drag the two ends to keep a part of the video. The cut is
  exact to the frame. Saved as `<name>-trimmed`.
- **Video Snapshots:** move to any frame (the Left and Right keys step one
  frame) and save it as a PNG picture, `<name>-snapshot.png`. The window
  stays open for more snapshots.
- **Split Video:** add split points on the timeline, drag them, or remove
  them with their ×. Every point starts a new part, cut exactly at that
  frame. The parts are saved in a new folder, `<name> - parts`.
- **Change Video Speed:** choose a speed from 0.25× to 4× while the preview
  plays at that speed. The sound keeps its natural pitch. Saved as
  `<name>-0.5x` (with the speed you chose).
  - **Smooth slow motion** (0.25×, 0.5× and 0.75×): creates new frames in
    between instead of repeating frames, so slow motion looks smooth. It
    takes much longer to process.
- **Crop Video:** drag the crop rectangle or its handles over the picture,
  choose a shape (Free, 1:1, 16:9, 9:16, 4:3, 3:4) or type the width and
  height in pixels. Saved as `<name>-cropped`.

A finished file never replaces an existing one: ` (1)`, ` (2)` … is added to
the name.

<p>
  <img src="docs/trim-en-light.png" width="49%" alt="The Trim Video window in English, light theme">
  <img src="docs/trim-en-dark.png" width="49%" alt="The Trim Video window in English, dark theme">
</p>

<p>
  <img src="docs/snapshots-en-light.png" width="49%" alt="The Video Snapshots window in English, light theme">
  <img src="docs/snapshots-en-dark.png" width="49%" alt="The Video Snapshots window in English, dark theme">
</p>

<p>
  <img src="docs/split-en-light.png" width="49%" alt="The Split Video window in English, light theme">
  <img src="docs/split-en-dark.png" width="49%" alt="The Split Video window in English, dark theme">
</p>

<p>
  <img src="docs/speed-en-light.png" width="49%" alt="The Change Video Speed window with Smooth slow motion in English, light theme">
  <img src="docs/speed-en-dark.png" width="49%" alt="The Change Video Speed window with Smooth slow motion in English, dark theme">
</p>

<p>
  <img src="docs/crop-en-light.png" width="49%" alt="The Crop Video window in English, light theme">
  <img src="docs/crop-en-dark.png" width="49%" alt="The Crop Video window in English, dark theme">
</p>

## Install

1. Download `FalakSetup-2.0.0.exe` from the
   [releases page](https://github.com/GlitchGLH/Falak/releases/latest).
2. Run it — no administrator rights needed; Falak installs for your user only
   (`%LOCALAPPDATA%\Programs\Falak`, or a folder you choose).
3. The Falak icon appears next to the clock. Drag a file and press Shift.

A portable version that needs no installation is attached to the release too.

[VirusTotal scan of the 1.0.0 installer](https://www.virustotal.com/gui/file/8a44dc4c52252f48a2559701b978cdb03fe30b2702f70fc931937c58c6e6da2c)

## Key-reading helper (optional)

When you drag from the desktop, Windows can hide the Shift key and the mouse
button from Falak. The optional helper (tray menu → Key-reading helper) reads
them with administrator rights, so Windows asks for your approval each time
the helper starts. Falak creates no scheduled task for it.

## SmartScreen

The first run may show a Windows SmartScreen warning because the installer is
not digitally signed (a code-signing certificate costs money every year).
Click **More info**, then **Run anyway**. Everything Falak does happens on
your computer — see Privacy below.

## Uninstall

Windows Settings → Apps → Falak → Uninstall (or `Uninstall.exe` in the install
folder). You are asked whether to delete your settings too; by default they
are kept. The uninstaller closes Falak, then removes the files it installed
(other files in a folder you chose are kept; the folder goes only when
empty), the shortcuts, the startup entry, the log folder and the downloaded
engines (`%LOCALAPPDATA%\Falak`), plus the helper's scheduled task left by
earlier builds, if any.

## Privacy

Everything happens on your computer. The only network access is the one-time
download of the free conversion engines when you first use a format that
needs one: FFmpeg, PDFium, qpdf and resvg, downloaded from their GitHub
release pages. Falak always fetches the same fixed releases (FFmpeg 9.0.2 by
gyan.dev, PDFium chromium/8138, qpdf 12.3.2, resvg 0.49.0) and checks each
download against its SHA-256 before using it. No telemetry, no analytics, no
uploads.

## License

MIT — see [LICENSE](LICENSE).

---

## فلك بالعربية

صمّمه وأشرف عليه UncleGlitch، وبُني بمساعدة الذكاء الاصطناعي.

**فلك** عجلة تحويل الملفات: اسحب ملفًا أو عدة ملفات، اضغط **Shift** أثناء
السحب، ثم أفلت على الصيغة التي تريدها — تُحفظ نسخة جديدة بجانب الملف الأصلي
ولا يتغير الأصل.

By UncleGlitch (GitHub: @GlitchGLH)

[تحميل أحدث إصدار](https://github.com/GlitchGLH/Falak/releases/latest)

<p>
  <img src="docs/wheel-light.png" width="49%" alt="The wheel, light theme">
  <img src="docs/wheel-dark.png" width="49%" alt="The wheel, dark theme">
</p>

<p>
  <img src="docs/tools-wheel-light.png" width="49%" alt="The tools wheel, light theme">
  <img src="docs/tools-wheel-dark.png" width="49%" alt="The tools wheel, dark theme">
</p>

<p>
  <img src="docs/tray-menu-ar-light.png" width="49%" alt="The tray menu in Arabic, light theme">
  <img src="docs/tray-menu-ar-dark.png" width="49%" alt="The tray menu in Arabic, dark theme">
</p>

<p>
  <img src="docs/compress-dialog-ar-light.png" width="49%" alt="The Compress dialog in Arabic, light theme">
  <img src="docs/compress-dialog-ar-dark.png" width="49%" alt="The Compress dialog in Arabic, dark theme">
</p>

<p>
  <img src="docs/progress-ar-light.png" width="49%" alt="The progress window in Arabic, light theme">
  <img src="docs/progress-ar-dark.png" width="49%" alt="The progress window in Arabic, dark theme">
</p>

<p>
  <img src="docs/how-to-use-ar-light.png" width="49%" alt="The How to use card in Arabic, light theme">
  <img src="docs/how-to-use-ar-dark.png" width="49%" alt="The How to use card in Arabic, dark theme">
</p>

### التثبيت

1. نزّل `FalakSetup-2.0.0.exe` من صفحة الإصدارات.
2. شغّله — لا يحتاج صلاحيات المسؤول، ويُثبَّت لحسابك فقط
   (`%LOCALAPPDATA%\Programs\Falak`).
3. يظهر رمز «فلك» بجوار الساعة. اسحب ملفًا واضغط Shift.

توجد أيضًا نسخة محمولة لا تحتاج تثبيتًا مرفقة بالإصدار.

### مساعد قراءة الأزرار (اختياري)

عند السحب من سطح المكتب قد يُخفي ويندوز عن «فلك» حالة زر Shift وزر الفأرة. يقرأ المساعد الاختياري (من قائمة الأيقونة ← مساعد قراءة الأزرار) هذه الحالة بصلاحيات المسؤول، لذلك يطلب ويندوز موافقتك في كل مرة يعمل فيها المساعد. لا يُنشئ «فلك» مهمة مجدولة له.

### SmartScreen

قد يعرض ويندوز تحذير SmartScreen عند أول تشغيل لأن المثبِّت غير موقَّع رقميًا
(شهادة التوقيع الرقمي مدفوعة سنويًا). اضغط **More info** ثم **Run anyway**.
كل ما يفعله فلك يجري على جهازك — انظر «الخصوصية» أدناه.

### إلغاء التثبيت

إعدادات ويندوز ← التطبيقات ← Falak ← إلغاء التثبيت (أو `Uninstall.exe` في
مجلد البرنامج). سيسألك إن كنت تريد حذف إعداداتك أيضًا، والافتراضي
الاحتفاظ بها. يُغلق إلغاء التثبيت «فلك» أولًا، ثم يزيل الملفات التي ثبّتها المثبّت، والاختصارات ومدخل بدء التشغيل ومجلد السجل والمحركات المنزَّلة، ومهمة المساعد المجدولة التي أنشأتها الإصدارات السابقة إن وُجدت. أما الملفات الأخرى التي وضعتها بنفسك في مجلد التثبيت فتبقى كما هي.

### الخصوصية

كل شيء يجري على جهازك. الاتصال بالإنترنت الوحيد هو تنزيل محركات التحويل
المجانية مرة واحدة عند أول استخدام للصيغ التي تحتاجها: FFmpeg وPDFium
وqpdf وresvg، من صفحات إصداراتها على GitHub. ويجلب «فلك» دائمًا الإصدارات الثابتة نفسها (FFmpeg 9.0.2 وPDFium chromium/8138 وqpdf 12.3.2 وresvg 0.49.0)، ويتحقق من بصمة SHA-256 لكل تنزيل قبل استخدامه. لا تتبّع ولا تحليلات ولا رفع
أي بيانات.

### الترخيص

MIT — انظر [LICENSE](LICENSE).
