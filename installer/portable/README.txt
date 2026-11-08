Falak 1.0.0 - portable
By UncleGlitch (GitHub: @GlitchGLH)
https://github.com/GlitchGLH/Falak

Run Falak.exe: drag a file (or several), press Shift while dragging, and drop on the format you want. A new file is saved next to the original.

On first run, Falak copies itself to %LOCALAPPDATA%\Programs\Falak (or to your existing Falak folder if it is already installed) and runs from there. Settings are kept in the registry (HKCU\Software\Falak).

When a conversion needs them, Falak downloads fixed versions of its engines (FFmpeg, PDFium, qpdf, resvg) into %LOCALAPPDATA%\Falak\tools and checks the SHA-256 of each download.

The optional key-reading helper asks for Windows approval (UAC) each time it starts.

To remove the portable copy: turn off "Start with Windows" in the tray menu, choose "Quit" in the tray menu, then delete these folders:
  %LOCALAPPDATA%\Programs\Falak
  %LOCALAPPDATA%\Falak
You can also delete the registry key HKCU\Software\Falak, which holds your settings.
If you installed Falak with the Setup, remove it from Settings > Apps instead.

----------------------------------------

فلك 1.0.0 - نسخة محمولة
By UncleGlitch (GitHub: @GlitchGLH)
https://github.com/GlitchGLH/Falak

شغّل Falak.exe: اسحب ملفًا أو عدة ملفات، واضغط Shift أثناء السحب، ثم أفلت على الصيغة التي تريدها. تُحفظ نسخة جديدة بجانب الملف الأصلي.

عند التشغيل الأول ينسخ «فلك» نفسه إلى %LOCALAPPDATA%\Programs\Falak (أو إلى مجلده الحالي إن كان مثبّتًا من قبل) ويعمل من هناك، ويحفظ إعداداته في السجل (HKCU\Software\Falak).

عند الحاجة يُنزّل «فلك» إصدارات ثابتة من محركاته (FFmpeg وPDFium وqpdf وresvg) إلى %LOCALAPPDATA%\Falak\tools، ويتحقق من بصمة SHA-256 لكل تنزيل.

يطلب مساعد قراءة الأزرار الاختياري موافقة ويندوز في كل مرة يعمل فيها.

لإزالة النسخة المحمولة: أوقف التشغيل مع بدء ويندوز من قائمة الأيقونة، ثم أغلق «فلك» منها، واحذف هذين المجلدين:
  %LOCALAPPDATA%\Programs\Falak
  %LOCALAPPDATA%\Falak
ويمكنك أيضًا حذف مفتاح السجل HKCU\Software\Falak الذي يحفظ إعداداتك.
وإن كنت ثبّتّ «فلك» بالمثبّت، فأزله من الإعدادات ← التطبيقات بدلًا من ذلك.
