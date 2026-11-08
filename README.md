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
| Archives | ZIP, 7Z, RAR, TAR, GZIP |

## Install

1. Download `FalakSetup-1.0.0.exe` from the
   [releases page](https://github.com/GlitchGLH/Falak/releases/latest).
2. Run it — no administrator rights needed; Falak installs for your user only
   (`%LOCALAPPDATA%\Programs\Falak`, or a folder you choose).
3. The Falak icon appears next to the clock. Drag a file and press Shift.

A portable version that needs no installation is attached to the release too.

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

1. نزّل `FalakSetup-1.0.0.exe` من صفحة الإصدارات.
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
