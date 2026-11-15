# Falak 2.0.0 — release notes

New video windows, image archives to PDF, a better way to compress many files,
and Falak's own message windows. Everything new since 1.0.0 (the 1.0.1 fix
included).

### New

- **Video windows** with a preview, opened from the tools wheel (one video at
  a time):
  - **Trim Video:** keep a part of the video, cut exactly to the frame.
  - **Video Snapshots:** save any frame as a PNG picture. It replaces the
    one-click snapshot.
  - **Split Video:** cut the video into parts at the points you choose. It
    replaces the old split in half.
  - **Change Video Speed:** 0.25× to 4× with a preview that plays at the
    chosen speed; the sound keeps its natural pitch. It replaces the fixed
    2× speed. **Smooth slow motion** (0.25×, 0.5×, 0.75×) creates new
    frames in between for smooth slow motion; it takes much longer.
  - **Crop Video:** a crop rectangle with handles, the shapes Free, 1:1,
    16:9, 9:16, 4:3 and 3:4, and the width and height in pixels.
- **Image archives to PDF:** a ZIP, RAR or 7Z that holds only images converts
  to one PDF, one image per page in natural order.
- **Compressing several files** saves the copies in a new
  `Compressed <date> <time>` folder next to the originals, with their
  original names.
- **Falak's own message windows** replace the Windows message boxes for
  errors and questions, in Falak's style and language (Ctrl+C copies the
  text).
- Popups open and close with the same smooth animation as the wheel.

### Fixed

- The wheel no longer stays closed on some computers where another program
  (such as a graphics overlay or an antivirus) made Windows report a busy
  state. Full-screen games and presentations still keep the wheel away
  (this was the 1.0.1 fix).
- Error messages show the real cause, such as "Access is denied.", instead of
  FFmpeg's progress line.
- In Arabic, the progress window shows the file name first, then the
  percentage.

Unsigned build: Windows SmartScreen may show a warning; choose More info, then Run anyway.

### SHA-256

```
<SHA-256 of Falak.exe — added after the final build>  Falak.exe
<SHA-256 of FalakSetup-2.0.0.exe — added after the final build>  FalakSetup-2.0.0.exe
<SHA-256 of Falak-2.0.0-portable.zip — added after the final build>  Falak-2.0.0-portable.zip
```

VirusTotal scan of FalakSetup-2.0.0.exe: <VirusTotal link — added after the scan>

## بالعربية

نوافذ فيديو جديدة، وتحويل أرشيف الصور إلى PDF، وطريقة أفضل لضغط عدة ملفات، ونوافذ رسائل خاصة بـ«فلك». كل الجديد منذ 1.0.0، ومعه إصلاح 1.0.1.

### الجديد

- **نوافذ فيديو** فيها معاينة، تُفتح من عجلة الأدوات (فيديو واحد في كل مرة):
  - **قص الفيديو:** احتفظ بجزء من الفيديو، بقص دقيق حتى مستوى الإطار.
  - **لقطات من الفيديو:** احفظ أي إطار صورة PNG. تحل محل اللقطة بنقرة واحدة.
  - **تقسيم الفيديو:** قسّم الفيديو إلى أجزاء عند النقاط التي تختارها. تحل محل التقسيم القديم إلى نصفين.
  - **تغيير سرعة الفيديو:** من 0.25× إلى 4× مع معاينة تعمل بالسرعة المختارة، ويحتفظ الصوت بطبقته الطبيعية. تحل محل السرعة الثابتة 2×. وخيار **«حركة بطيئة ناعمة»** (0.25× و0.5× و0.75×) يُنشئ إطارات جديدة بين الإطارات لحركة بطيئة ناعمة، لكنه يستغرق وقتًا أطول بكثير.
  - **اقتصاص الفيديو:** مستطيل اقتصاص بمقابض، والأشكال حرّة و1:1 و16:9 و9:16 و4:3 و3:4، والعرض والارتفاع بالبكسل.
- **أرشيف الصور إلى PDF:** ملف ZIP أو RAR أو 7Z يحتوي صورًا فقط يتحوّل إلى ملف PDF واحد، كل صورة في صفحة، بترتيب طبيعي.
- **ضغط عدة ملفات** يحفظ النسخ في مجلد جديد `Compressed <التاريخ> <الوقت>` بجانب الملفات الأصلية، بأسمائها الأصلية.
- **نوافذ رسائل خاصة بـ«فلك»** تحل محل نوافذ رسائل ويندوز في الأخطاء والأسئلة، بأسلوب «فلك» ولغته (Ctrl+C ينسخ النص).
- النوافذ المنبثقة تظهر وتختفي بنفس حركة العجلة الناعمة.

### الإصلاحات

- لم تعد العجلة تبقى مغلقة على بعض الأجهزة التي يجعل فيها برنامج آخر (مثل طبقة عرض لبطاقة الرسوميات أو برنامج حماية) ويندوز يُبلغ بحالة انشغال. وما زالت الألعاب والعروض التقديمية بملء الشاشة تمنع ظهور العجلة (هذا إصلاح 1.0.1).
- رسائل الخطأ تعرض السبب الحقيقي، مثل "Access is denied."، بدل سطر التقدم من FFmpeg.
- في الواجهة العربية، تعرض نافذة التقدم اسم الملف أولًا ثم النسبة المئوية.

نسخة غير موقّعة: قد يعرض Windows SmartScreen تحذيرًا؛ اختر More info ثم Run anyway.

---

# Falak 1.0.0 — release notes

First stable release.

- The wheel: drag a file, press Shift, drop on a format. Images, PDF, Office
  documents, video, audio, subtitles, text files and archives.
- The wheel sounds play through one persistent output stream — crisp ticks
  that never fade on Bluetooth or after sleep.
- The approved Falak icon on the exe, the tray icon and the menu header.
- Per-user installer (no admin prompt) with Arabic and English, automatic
  closing of a running copy, and an uninstaller that removes everything.

Unsigned build: Windows SmartScreen may show a warning; choose More info, then Run anyway.

### SHA-256

```
41374c553b7e3a0e11ba5e179f1b4a424a33fa5385667096f867ef970a1c1d7c  Falak.exe
8a44dc4c52252f48a2559701b978cdb03fe30b2702f70fc931937c58c6e6da2c  FalakSetup-1.0.0.exe
a42d03a2cc3bb0f7c39f0887954cf05a4b716392e3d9eb8883c645e95d2d5926  Falak-1.0.0-portable.zip
```

VirusTotal scan of FalakSetup-1.0.0.exe: https://www.virustotal.com/gui/file/8a44dc4c52252f48a2559701b978cdb03fe30b2702f70fc931937c58c6e6da2c

## بالعربية

أول إصدار مستقر من «فلك».

- العجلة: اسحب ملفًا، اضغط Shift، وأفلت على الصيغة. صور وPDF ومستندات أوفيس
  وفيديو وصوت وترجمات وملفات نصية وأرشيفات.
- أصوات العجلة الآن عبر تيار صوتي ثابت واحد: نقرة واضحة لا تخفت على
  البلوتوث أو بعد السكون.
- أيقونة فلك المعتمدة على الملف التنفيذي وأيقونة شريط المهام ورأس القائمة.
- مثبِّت لحساب المستخدم (بدون صلاحيات مسؤول) بالعربية والإنجليزية، يغلق
  النسخة العاملة تلقائيًا، وإلغاء تثبيت ينظف كل شيء.

By UncleGlitch — https://github.com/GlitchGLH/Falak
