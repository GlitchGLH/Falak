# CLAUDE_HANDOVER — Falak 1.0.0

## Current state (latest session)

Written at the end of the 2026-11-08/09 review session. This section wins over anything
older below it. Arabic is quoted character for character.

### Where things are
- Repository: `C:\Projects\Falak`, git branch `main`, local config `core.autocrlf=false`,
  `user.name=UncleGlitch`, `user.email=GlitchGLH@users.noreply.github.com` (local only; the
  owner may change it). History starts at `8816700` (import of the handed-over state). Nothing
  has been pushed to GitHub.
- HEAD before this handover commit: `7e0b4eb`. Working tree clean.
- Version: still **1.0.0** everywhere.
- Release folders (Microsoft false-positive review pending on their hashes): never modify;
  read only with the owner's explicit permission:
  - `C:\Users\Glitch\Desktop\Falak-1.0.0-release`
  - `C:\Users\Glitch\Desktop\Falak with Claude Code\Falak-1.0.0-release`

### Latest build/ outputs (built from `7e0b4eb`, vcs.modified=false; not published)
| File | SHA-256 |
|---|---|
| `build/Falak.exe` | `bfd0e80fd3a0472dad623f3a0844564ecb95d63e52c03e4607870c6998d70e9c` |
| `build/FalakSetup-1.0.0.exe` | `811a2371910d00c996ae3e552d80fda842dfc935dbb0e4d8dd4ed4ea230cd9d8` |
| `build/Falak-1.0.0-portable.zip` | `47668cdebe00e9c178432d51e67cad5b52593583f9d2670023bd863d0797471b` |

Every commit changes these hashes (Go embeds the git revision). Do not rebuild until the
owner asks (NEXT item 5 is the final build).

### Commits since the import
| Commit | Title | Purpose |
|---|---|---|
| `8816700` | import current state into git | First commit of the handed-over 1.0.0 source (old history is not in this repo). |
| `a0456db` | uninstaller reads the keep-settings answer correctly | `MessageBox … IDYES` jump instead of `Pop $0` (it read a leftover nsExec code); every nsExec result popped. |
| `d4b9569` | uninstaller removes only the files it installed | No `RMDir /r "$INSTDIR"`; deletes its own files, `RMDir` only when empty; audit check. |
| `b3a7987` | video compress no longer blocks other conversions | `engineMu` held only for the FFmpeg download, not the whole encode. |
| `7565407` | handover HEIC notes match the code | HEIC input = gen2brain/heic (pure-Rust crate, WASM→Go); output = FFmpeg libx265 + own HEIF container. |
| `79b7d5c` | video compress-to-size works with current FFmpeg | `-pass:v` / `-passlogfile:v` (plain `-pass 2` hit the AAC stream). |
| `3db401c` | compress leaves no two-pass logs in .falak-tmp | Removed `-0.log*` logs; test checks `.falak-tmp` is gone. |
| `f6eedbc` | GIF progress test uses a 9 s clip | Test was timing-flaky on fast machines. |
| `4901549` | key helper asks UAC each time; no scheduled task | Removed silent-elevation task `Falak\KeysHelper`; helper/installer/uninstaller delete an old one. |
| `cf15b37` | NSIS script compiles with makensis on Windows | `${__FILEDIR__}\…` backslashes (Windows makensis 3.13 failed on `/`); audit check. |
| `0f3acfe` | uninstaller without PowerShell; Falak closed by message | NSIS `EnumRegValue` loop for Layers; `wmQuitUpdate` (0x8009) to `FalakMsg`, wait 5 s, then `taskkill /F`, +3 s for the helper. |
| `14462d5` | helper prompt says approval is asked each time | Owner text: EN "Windows will ask for your approval each time the helper starts." AR «سيطلب ويندوز موافقتك في كل مرة يعمل فيها المساعد.» |
| `9821259` | engines pinned to fixed releases with SHA-256 checks | `conv/engines.go`, `falak-engine.txt`, hash check, `enginesnet` test. |
| `b32152d` | engines live in %LOCALAPPDATA%\Falak\tools | Engines out of the install folder; old `Programs\Falak\tools` removed. |
| `6081f23` | one install folder when the user picks their own | `installDir()` = installer's InstallLocation; old second copy removed; `InstallDirRegKey`. |
| `c311c4e` | vendor the dependencies; builds from a fresh clone | `vendor/` + `go.sum`, no `../deps`; all deps proven unmodified upstream; 3 version replaces; `godebug default=go1.24`; test images in `conv/testdata`. |
| `bd93a63` | docs match the review changes | README (English part), notices, licenses (x/sync dropped), handovers. |
| `50cdd9d` | CLAUDE.md with the permanent rules | Rule file at repo root. |
| `48618a9` | Arabic README matches the review changes | Owner's Arabic: uninstall sentence, helper section, pinned-engines sentence. |
| `3967648` | portable README.txt in the repo | `installer/portable/README.txt`, owner's EN+AR text. |
| `d8c6433` | build step for the portable zip | `installer/make_portable.py` (deterministic zip of exactly 3 files) + audit check. |
| `1dbcfbd` | CLAUDE.md release-folder rule: read only with permission | Both release-folder paths listed. |
| `4fc1ea2` | compress to a set size works in watched folders; failures shown | Pass logs in `%TEMP%\falak-pass-*`; `completePassLogs` renames x264's `*.temp` logs with retry; FFmpeg errors logged (args + last 15 stderr lines) and the specific line shown; failed jobs also show a message box. |
| `7516d74` | the target size box can be focused and edited | Focus, caret, selection, Enter; default 100 MB or half the original (≤ 100 MB). |
| `bc37f7d` | no leftovers in .falak-tmp after failed or interrupted jobs | Delete/move retry 5 s; `temp-folders.txt` record; start-up cleanup of stale `<UnixNano>-<n>…` entries only. |
| `8ea1363` | helper menu item shows whether the helper really runs | Toggle = helper running; refusing UAC turns the setting off. |
| `a52b66c` | once-per-session hint when a desktop drag needs the helper | Owner text (see below), tray balloon. |
| `1927fab` | log every helper hint/offer attempt and tray notification | Logs Shell_NotifyIcon result + notification settings (read only). |
| `0b40241` | helper hint also shown in a Falak window | Message box in addition to the balloon (balloons are silenced on the owner's machine). |
| `06b34cd` | first run also opens the How to use card | Only on the very first run, never with `--startup`. |
| `7e0b4eb` | background job done: the progress window shows Done briefly | 2.5 s, no focus, X / new drag close it, not over the wheel or another panel. |

UX 5 hint texts (owner-approved, in `helper_windows.go`):
- Title EN "Desktop drags need the helper" / AR «السحب من سطح المكتب يحتاج المساعد»
- Body EN "Falak can't read the Shift key when dragging from the desktop unless the Key-reading helper is on. Turn it on from the Falak icon next to the clock."
- Body AR «لا يستطيع «فلك» قراءة زر Shift عند السحب من سطح المكتب إلا بتشغيل «مساعد قراءة الأزرار». شغّله من أيقونة «فلك» بجانب الساعة.»

### Pinned engines
Stored in `%LOCALAPPDATA%\Falak\tools\<engine>\`. Defined in `conv/engines.go` (URL +
SHA-256 of the archive). Each download is hashed while it is written; a mismatch deletes it
and reports the existing "The download didn't finish" / «التنزيل لم يكتمل» inside the
engine's existing "Couldn't download …" message. Each engine folder gets `falak-engine.txt`
(hash + URL); a folder without the current pin is downloaded again.
| Engine | Release | Archive SHA-256 |
|---|---|---|
| FFmpeg | 9.0.2 `full_build-shared`, GyanD/codexffmpeg | `8d31e162f1616e37aab3fa2db991b97e1b8dbeb1c8465fd81a81be16ff91b328` |
| PDFium | chromium/8138, bblanchon/pdfium-binaries | `8d48973e3eb472dda0f5f2435e595ac043f9f2219dd71883392f549445afa8a5` |
| qpdf | 12.3.2 mingw64 | `ebeb4692434a8cb8a27f876584c0c2b402b6251dc6c8ed4b39cf2304cada10e0` |
| resvg | 0.49.0 win-x86_64 | `38af9631b37e61118a32a5ef31e6be6af083eda63b4c17ffc4fdc63e449277df` |
After a pin change: `go test -tags enginesnet -run TestEnginePins -v ./conv`.

### Verified by the owner on real Windows
- Upgrade from the old release: the old `Falak\KeysHelper` task was removed, the old helper
  closed, the old `%LOCALAPPDATA%\Programs\Falak\tools` folder removed, Falak closed by the
  installer.
- Helper: UAC is asked on each start; refusing turns the setting off and the menu item goes
  grey; enabling it from the menu and pressing Yes brings the wheel back for desktop drags.
- All four engines downloaded into `%LOCALAPPDATA%\Falak\tools` (FFmpeg 9.0.2, PDFium
  chromium/8138, qpdf 12.3.2, resvg 0.49.0). MP4→MP3, MP4→MKV, PDF→PNG, PDF compress and
  SVG→PNG work.
- Compress to a set size from `G:\NVIDIA\Desktop` at 60 MB and 40 MB completed, with the
  editable size field (caret, focus border), and no `.falak-tmp` was left behind.

### Owner's machine notes
- Notifications are disabled by Chris Titus Tech WinUtil:
  `HKCU\Software\Policies\Microsoft\Windows\Explorer\DisableNotificationCenter = 1` and
  `HKCU\Software\Microsoft\Windows\CurrentVersion\PushNotifications\ToastEnabled = 0`.
  Tray balloons never show here: never rely on balloons alone for anything the user must see.
  **Never change the owner's Windows settings** (read only, as `notificationState()` does).
- `HKCU\Software\Falak` already has `Welcomed`, so the next start is not a first run (see NEXT 1).
- Defender real-time protection is on; Controlled Folder Access is off.
- Drive G: (NTFS, "My Archive") holds NVIDIA recordings in `G:\NVIDIA\Desktop`; programs
  there briefly lock new files.

### DONE
- Code review findings fixed (uninstaller, compress lock, helper elevation, PowerShell,
  pinned engines, one install folder, vendoring, temp leftovers, compress in watched
  folders, editable size box, helper state/hint, first-run card, background "Done").
- Docs: README (EN + owner's AR), notices, licenses, handovers, CLAUDE.md, portable README.txt.
- Owner's manual tests listed above.

### IN PROGRESS
- Nothing in progress in code. The last build (`7e0b4eb`) is waiting for NEXT 1 and 2.

### NEXT (in this order)
1. Uninstall test answering «لا» (settings must be deleted, including `Welcomed`). Then
   install the new build and run Falak from the installer's finish page: the "How to use"
   card must open (`06b34cd`). Then restart Falak and confirm the card does NOT open again,
   and does not open with "Start with Windows".
2. Manual test of the helper hint message box (`0b40241`) and the progress window shown again
   for 2-3 seconds with «اكتمل التحويل» after a background job (`7e0b4eb`): no focus
   stealing, no new sound, X and a new drag close it.
3. Custom folder `C:\FalakTest` with `keep.txt`, answering «نعم» (settings kept, `keep.txt` kept).
4. Screenshots for GitHub (English and Arabic, Light and Dark). Arabic screenshots need the
   owner's approval.
5. Final commit and final build. No more builds after that.
6. New VirusTotal scan and new Microsoft false-positive report on the final
   `FalakSetup-1.0.0.exe`.
7. GitHub release v1.0.0 (account GlitchGLH, repo Falak, author UncleGlitch, MIT license).

### Rejected by the owner (do not implement)
- Proposal 3: on a manual start that is not the first run, open the tray menu or the How to
  use card in addition to the «فلك يعمل في الخلفية» balloon. **Rejected.**
- Proposal 4: replace or add to the «يستمر التحويل في الخلفية» balloon (job sent to the
  background) with a window. **Rejected; it stays a balloon only.**

### Known and deferred (owner's decision 2026-11-09: do NOT fix for 1.0.0)
- **Portable downgrade:** running an older portable copy can downgrade the installed Falak
  (as found in the first code review). Known; not fixed.
- **wmQuitUpdate:** any local program can close Falak by sending `wmQuitUpdate` (0x8009) to
  the `FalakMsg` window (the uninstaller/installer use it to close Falak). Known; not fixed.

### Open questions (word for word, not yet answered by the owner)
1. "This local repository's history starts at 8816700, while github.com/GlitchGLH/Falak has the
   old history. How do you want to publish: push this history to a new branch, replace main,
   or upload files as before?"
2. "The local git identity is UncleGlitch <GlitchGLH@users.noreply.github.com>. Is that the
   name and email you want on the commits?"
3. "The vendored dependencies make the repository about 47 MB (clone about 61 MB) instead of
   about 2.6 MB. Is that acceptable for the public repository?"
4. "Compress to a set size computes MB as 1,048,576 bytes (a 100 MB target gave 103.3 MB =
   98.5 MB in Explorer), while the size check uses 1,000,000 bytes. Keep it, or align both?"
5. "Not fixed from the first review (low priority): a finished file can replace a file of the
   same name created during a long conversion (os.Rename replaces); running an older portable
   copy downgrades the installed Falak; any local program can close Falak with wmQuitUpdate.
   Fix any of these before the release?"
6. "The old deps folder still sits in C:\Projects\Falak\deps (322 MB, gitignored, no longer
   used since c311c4e). May it be deleted?"

### Environment
- Go **go1.27.3 windows/amd64** (`C:\Program Files\Go`), installed with winget
  (`GoLang.Go`). Module says `go 1.25.0` with `godebug default=go1.24`.
- NSIS **3.13** (`C:\Program Files (x86)\NSIS`), installed with winget (`NSIS.NSIS`).
- Python **3.14.10**, run as `py` (`python` is the Store alias and does not work in Git Bash).
- go-winres not installed (the `.syso` is checked in).
- `C:\Projects\deps` junction: created for the first test run, **removed after vendoring**
  (`c311c4e`); builds and tests no longer need it. The old copies remain in
  `C:\Projects\Falak\deps` (gitignored, unused).
- FFmpeg for tests: **`C:\Projects\tools\ffmpeg-9.0.2`** (the pinned 9.0.2, outside the
  repo; put it on PATH; FFmpeg tests skip without one). The older `C:\Projects\tools\ffmpeg`
  (BtbN master) was deleted at the owner's request.
- Shell env used for every gate (Git Bash):
  `export PATH="/c/Program Files/Go/bin:/c/Program Files (x86)/NSIS:/c/Projects/tools/ffmpeg-9.0.2:$PATH"`
  and `GOFLAGS= GOPROXY=off GOSUMDB=off GOMEMLIMIT=2500MiB`.
- Gates: `go test -count=1 ./...`; `GOOS=linux go vet -tags "wasm2go nodynamic" ./...`
  (clean); `go vet -tags "wasm2go nodynamic" ./...` on Windows prints **12 pre-existing
  "possible misuse of unsafe.Pointer" warnings** (pdfium_windows.go, wic_windows.go,
  droptarget_windows.go, main_windows.go, menu_windows.go) — not the gate;
  `py installer/audit_paths.py`.
- `gofmt -l` flags **one pre-existing formatting line in `main_windows.go`** (cosmetic, never
  part of the gate; do not reformat the file, it would touch sound call-site lines).
- Builds, only into `build/`:
  1. `GOOS=windows GOARCH=amd64 go build -p 1 -tags "wasm2go nodynamic" -ldflags "-s -w -H windowsgui" -trimpath -o build/Falak.exe .`
  2. Setup: copy `installer/falak.nsi`, `raha-nsis.ico`, `license.txt` and `build/Falak.exe`
     into `build/nsis/installer/`, `THIRD_PARTY_LICENSES.txt` into `build/nsis/`, run
     `makensis -INPUTCHARSET UTF8 C:\Projects\Falak\build\nsis\installer\falak.nsi`
     (output `build/FalakSetup-1.0.0.exe`), then delete `build/nsis`.
  3. `py installer/make_portable.py` → `build/Falak-1.0.0-portable.zip`.
- Logs on the owner's machine: `%LOCALAPPDATA%\Falak\falak.log` (readable; very useful).

### Mistakes and dead ends (do not repeat)
- Python edits through a bash heredoc mangled escapes: `\n\n` became real line breaks inside a
  Go string, and `\t` / `\f` in Windows paths became a tab and a form feed. Write the script
  to a file with raw strings (or edit with the Edit tool) and re-read the result.
- Python `write_text` on Windows writes CRLF; write bytes to keep LF. `grep -c $'\r'` in Git
  Bash miscounts; check CR bytes with Python.
- `${__FILEDIR__}/…` in NSIS: works on Linux makensis, fails on Windows ("no files found").
- x264 pass-1 logs in `G:\NVIDIA\Desktop` (or any folder another program watches) can stay
  locked as `*.temp`; FFmpeg still exits 0. That is why pass logs live in `%TEMP%` and
  `completePassLogs` exists. A reproduction in another folder passes — test in the real folder.
- Tray balloons silently do not show when notifications are off; `Shell_NotifyIcon` still
  returns success. Anything the user must see needs a Falak window too.
- `audit_paths.py` once matched its own comment ("PowerShell"); the check now ignores
  comment lines. Read gate output before committing.
- `go vet` with `GOFLAGS=-mod=mod` stopped working after vendoring; use empty `GOFLAGS`.
- `pdfium.dll` stays loaded in a test process; `t.TempDir()` cleanup fails on Windows.
- Recursive deletes outside the repo (e.g. on G:) are blocked by Claude Code's safety check;
  ask the owner to delete instead of working around it.
- The task list tool counts and the `git commit` exit codes hide failures in pipelines;
  check each gate's output, not just the last line.

---

**READ THIS FIRST (new session):** You are taking over Falak, a finished, released-quality
Windows drag-and-drop file converter, at version **1.0.0**. Two things are in flight and
must not be disturbed: a **Microsoft antivirus false-positive review** and a **not-yet-published
GitHub release**. **Never touch `Falak-1.0.0-release`** (the shipped binaries whose hashes
are under review). New builds are allowed but go to `build/` only (gitignored), never over
the release files. Everything else (code review, fixes, new tasks, screenshots, docs)
proceeds normally under the rules in Section 4 and `CLAUDE.md`.

---

## 1. What Falak is

Falak (فلك — "orbit/celestial sphere", internal Go module name `raha`) is a **drag-and-drop
conversion wheel for Windows 10/11 (x64)**. Drag any file with **Shift held**, and a frosted-glass
radial wheel pops up around the cursor. Each slice is one target format or tool; drop on a slice
to convert. It is fully bilingual (**English / Arabic with RTL mirroring**), light/dark aware,
renders its whole UI itself (no system controls), and converts with open engines: FFmpeg,
PDFium, qpdf and resvg are downloaded on demand to `%LOCALAPPDATA%\Falak\tools`; libavif/aom/dav1d,
libwebp and the pure-Rust `heic` decoder crate are embedded (WASM, transpiled to Go). Author: **UncleGlitch**; license **MIT**;
repo **github.com/GlitchGLH/Falak** (public).

### Features
- **Shift-drag wheel**: pop-open entrance (scale 40%→100%, exponential, ~30 ms), per-segment
  hover with critically-damped spring colour fade **and hover lift** (1.055 scale + 3.6 logical px
  push, blurred-silhouette shadow, 240 Hz sub-stepped springs), entrance label fade, ring
  cross-fade with rotation between format and tool sets.
- **Live glass**: the wheel blurs the real desktop behind it (Gaussian, radius 80 logical px),
  refract "watery" strip and segments, glints; solid-glass popups elsewhere (see decision history).
- **Center stays open**: Windows' own drag image shows the file through the hole; a frosted
  capsule shows file size (or hovered option, or progress %).
- **Progress window**: appears while converting (progress %, thin blue bar, drag-to-move);
  background conversions keep going and notify when done.
- **Tray icon + menu**, startup with Windows, single-instance, Arabic UI mirror, dark/light
  theme following Windows, **wheel sounds** (see SOUND RULE), elevation helper.
- **Compress dialog**: drop on "Compress" opens a real options dialog (target size in MB,
  red hint «اكتب حجمًا أصغر من حجم الملف الأصلي» when the target is not smaller than the
  original; Start activates only for `0 < MB` and `MB×1e6 < OriginalBytes`).
- **How-to-use card**: an in-app help card in the app's own style (no browser).
- **GIF limit**: animated-GIF output refuses videos longer than 10 s
  («صيغة GIF تدعم الفيديو حتى 10 ثوانٍ فقط») and reports two-pass progress (0 → 0.4 palette pass → 1.0).
- **Key-reading helper**: opt-in elevated helper so key state can be read while an elevated
  window has focus. It starts through a **UAC prompt every time** (`ShellExecute "runas"`);
  the old scheduled task `Falak\KeysHelper` (silent elevation of a user-writable exe) is gone
  and is deleted by the elevated helper, the installer and the uninstaller when found.

### Conversions per format (see README.md's table for the authoritative matrix)
- **Images**: between JPG, PNG, WEBP, GIF, AVIF, TIFF, BMP, ICO, HEIC (input and output), plus **SVG**
  (render via resvg) and **RAW** (read-only input) — targets are filtered per source kind.
- **PDF**: images→PDF (page-fitted), PDF→PNG/JPG, PDF→TXT; a JPG/PNG can become a **DOCX**
  (Word file with the picture page-fitted — no office suite needed).
- **Office/text**: DOCX→PDF/TXT, TXT→JPG/PNG, subtitle text formats — **SRT/VTT/ASS/TXT**
  conversions (commit 732f9e8 "format table matches Tangerine").
- **Audio**: between MP3, WAV, M4A, FLAC, OGG, AAC (FFmpeg); per-format tools
  (volume/speed/extract etc. as listed by `conv.ToolItemsFor`).
- **Video**: between MP4, WEBM, MOV, AVI, GIF; animated-GIF path has the 10 s cap and
  palette-based two-pass quality; per-format tools (rotate/trim/mute/speed as listed).
- **Archives**: 7Z/GZIP/RAR offered for compressible files (extract supported for archives
  via `archiveTools`).

### Tools (Alt menu — `conv.ToolItemsFor`, gated per file kind)
COMPRESS, METADATA, ROTATE, RESIZE 50%, WHITE BG, BLACK BG (+ per-category tool sets for
video/audio/pdf). Items that don't apply to the dragged file are shown disabled, never hidden.
Hovering a tool segment switches the whole wheel to the tools set (Alt/Ctrl/Tab also switch).

### Tray menu (exact strings from source, never retype)
- {ID: "theme", Title: T("Dark theme", "الثيم الداكن"), Sub: T("Light"/"Dark" → "فاتح"/"داكن")}
- {ID: "sound", Title: T("Wheel sounds", "أصوات العجلة"), Sub: T("Hover and drop sounds", "صوت عند التمرير والإفلات")}
- {ID: "startup", Title: T("Start with Windows", "التشغيل مع بدء ويندوز"), Sub: T("Runs when you sign in", "يعمل عند تسجيل الدخول")}
- {ID: "helper", Title: T("Key-reading helper", "مساعد قراءة الأزرار"), Sub: T("Administrator", "بصلاحيات المسؤول")}
- {ID: "lang", Title: "العربية", Sub: T("Arabic interface", "واجهة عربية")}
- {ID: "help", Title: T("How to use", "طريقة الاستخدام")} · {ID: "quit", Title: T("Quit", "إنهاء")}

Other exact Arabic strings you may need (from source):
- Compress hint: «اكتب حجمًا أصغر من حجم الملف الأصلي»
- GIF refusal: «صيغة GIF تدعم الفيديو حتى 10 ثوانٍ فقط»
- T('Falak is running', 'فلك يعمل الآن') · T('Drag a file and press Shift while dragging.', 'اسحب ملفًا واضغط Shift أثناء السحب.')
- T('Conversion failed', 'تعذّر التحويل') · T('Done', 'اكتمل التحويل') · T('Compressing', 'جارٍ الضغط')
- T('Still converting in the background', 'يستمر التحويل في الخلفية')
- T("You'll get a notification when it's done. Follow the progress on the Falak icon next to the clock.", 'سيظهر إشعار عند الانتهاء. تابع التقدّم من أيقونة فلك بجانب الساعة.')
- T("This option doesn't apply to the selected files", 'هذا الخيار لا ينطبق على الملفات المحددة')
- T('%s — %d file(s)', '%s — %d ملف') · T('%d files', '%d ملفات')
- README credit (AR): «صمّمه وأشرف عليه UncleGlitch، وبُني بمساعدة الذكاء الاصطناعي.»

**Rule: never change any existing Arabic string. New Arabic only when the user provides the
exact text, shown in full in your reply.**

### Sounds
Three bundled WAVs (`res/hover.wav` 27 ms, `res/appear.wav` 100 ms, `res/drop.wav` 110 ms,
mono 16-bit PCM 48 kHz) played through **one persistent winmm `waveOut` stream** with a custom
small mixer (`sound_windows.go`) — hover loops softly while the wheel is visible
(`soundKeepAlive`), appear/drop are one-shots. **Do not touch any of it** (Section 5).

## 2. File map (own files; plus `vendor/`, see Section 3)

**Root docs/config**: `README.md` (EN/AR product page, format table, credit lines) ·
`BUILDING.md` (build steps) · `HANDOVER.md` (short internal handover: paths, commands, rules) ·
`CLAUDE_HANDOVER.md` (this file) · `LICENSE` (MIT) · `RELEASE_NOTES.md` ·
`THIRD_PARTY_LICENSES.txt` (25 Go modules + wazero NOTICE + libavif/aom/dav1d/libwebp texts;
ships in installer + portable) · `THIRD_PARTY_NOTICES.md` · `CLAUDE.md` (permanent rules) ·
`go.mod`/`go.sum` (module `raha`, upstream versions, 3 version replaces) · `vendor/` ·
`.gitignore` (ignores `*.exe`, `build/`, the old local `deps/`) · `rsrc_windows_amd64.syso`
(checked-in icon+VERSIONINFO 1.0.0.0+manifest, linked automatically).

**App entry/Windows layer**: `main_windows.go` (App lifecycle, drag sessions, tick/present,
capture-free popups, tray init, `logAffinity`) · `droptarget_windows.go` (COM drag&drop
target) · `tray_windows.go` (tray icon) · `menu_windows.go` (tray menu window, rows, actions,
fade/slide animation via `presentLayeredA`) · `compdialog_windows.go` (Compress dialog host,
no captures) · `help_windows.go` (How-to-use host) · `win32_windows.go` (Win32 decls) ·
`input_windows.go` (keyboard state) · `elevation_windows.go` (run-as-admin) ·
`helper_windows.go` (key helper opt-in, UAC each start) · `install_windows.go` (self-install
into `installDir()` = installer's InstallLocation or the default; old helper task removal) ·
`startup_windows.go` (Run key) · `theme_windows.go` (dark mode detect) · `i18n_windows.go`
(`T(en, ar)`) · `sound_windows.go` (**FROZEN**) · `sound_windows_test.go` · `log_windows.go`.

**`ui/` (all self-drawn UI)**: `wheel.go` (geometry constants: LogicalSize 360, rBase 156,
rOut 147, rIn 60.5, rHole 52.3; render pipeline; hover springs; `segmentPath`;
`segShadow`/`drawSegShadow`; `drawLiftedSegment`) · `menu.go` (tray menu model/draw, card
colors dark {40,40,44} / light {255,255,255}) · `panel.go` (progress panel, solid `glassBase`) ·
`compdialog.go` (Compress dialog model: `TargetMB`, `TargetTooBig`, `OriginalBytes`,
`StartActive`) · `helpcard.go` · `icons.go` (supersampled line icons) · `fonts.go` (Inter/Amiri)
· `blur.go` (FrostBackground, GaussianBlur, box blurs) · `zoom.go` · `textblock.go` ·
tests: `lift_test.go`, `seam_test.go` (hover-seam regression), `menu_test.go`, `visual_test.go`,
`race_test.go`, `glass_bench_test.go`.

**`conv/` (engines/conversions)**: `conv.go` (items per category, dispatch) · `formats.go`
(format tables) · `media.go` (image/video via FFmpeg, `videoToGIF`+`gifPalettePass`, 10 s cap,
compress-to-size two-pass) · `gif_test.go` · `target_test.go` · `heif.go` (HEIC **output**:
FFmpeg `libx265` encodes HEVC, `heif.go` writes the HEIF container itself) · `imgdocx.go` (JPG/PNG→DOCX) · `office.go`+`office_windows.go` (DOCX) · `pdf.go` ·
`pdfium_windows.go`+`pdfium_other.go` (PDFium) · `qpdf.go` · `svg.go` (resvg) · `raw.go` ·
`merge.go` · `metadata.go` · `sevenzip_write.go` · `text.go` · `textarchive.go` ·
`wic_windows.go` (WIC codecs) · `lang.go` · `progress.go`(+test) · `tempfiles.go` ·
`conv_test.go`/`media_test.go`/`text_test.go` · `engines.go` (pinned engine URLs + SHA-256,
`falak-engine.txt` pin file) · `engines_test.go` (offline) · `engines_net_test.go` (tag
`enginesnet`: downloads and runs all four engines) · `testdata/` (sample HEIC/AVIF/WEBP).

**`cmd/`** (dev-only preview/bench tools): `preview`, `themeprev`, `menuprev`, `headerprev`,
`compressprev`, `helpprev`, `panelprev`, `hoverprev`, `entrance`, `toolprev`, `iconprev`,
`icons`, `pillcrop`, `opacity`, `bench`, `canceltest`, `ptime`, `qtest`, `t`, `pdfdl`,
`pdftest`, `pdftools`, `wictest`.

**`installer/`**: `falak.nsi` (NSIS; backslash runtime paths; full registry block; desktop
+ Start Menu shortcuts with explicit icon; ships THIRD_PARTY_LICENSES.txt; `BrandingText
"Falak 1.0.0 · By UncleGlitch"`; OutFile two levels above `installer/`) · `license.txt` ·
`raha.ico`/`raha-nsis.ico` · `audit_paths.py` (static NSIS path audit — part of the gate) ·
`verify-install.ps1` (real-Windows runtime verification) · `Falak.exe` (gitignored build drop) ·
`make_portable.py` + `portable/README.txt` (portable zip, see Section 3).
Compile-time asset paths use backslashes (`${__FILEDIR__}\...`): makensis on Windows does not
find `${__FILEDIR__}/...` files.

**`res/`** (**FROZEN sounds** `hover.wav`/`appear.wav`/`drop.wav`; `raha.ico`, `raha.rc`,
`raha.manifest`, `icon-1024.png`).

**`docs/`**: `wheel-light-dark.png`, `tray-menu.png`, `compress-dialog.png` — see Section 6.

## 3. Build environment (exact) and commands

| Tool | Version |
|---|---|
| Go | **go1.27.3** windows/amd64 (module says `go 1.25.0`) |
| NSIS (makensis) | **v3.13** (Windows) |
| Python | **3.14** (`py`; used for audits) |
| go-winres | not installed — the `.syso` is prebuilt and checked in |

Machine (Windows 11): repo at `C:\Projects\Falak` (git, branch `main`, local config
`core.autocrlf=false`). Test-only FFmpeg 9.0.2 at `C:\Projects\tools\ffmpeg-9.0.2` (outside the
repo; put it on PATH for the FFmpeg tests, which skip without one).

**Dependencies are vendored** (`vendor/`, `go.sum`): a fresh clone builds offline. Every module
is unmodified upstream code (checked file by file against the Go module proxy when the old
local `deps/` copies were retired — commit `c311c4e`). The former "v0.0.0" pins
(gen2brain/avif|heic|webp, srwiley/oksvg|rasterx) turned out to be plain upstream code.
`replace` pins three modules to the versions Falak 1.0.0 is built from, because other modules
ask for newer ones: `golang.org/x/sys v0.30.0`, `ebitengine/purego v0.8.4`,
`andybalholm/brotli v1.1.0`. `godebug default=go1.24` keeps the 1.0.0 GODEBUG defaults.
To change a dependency: edit go.mod → `go mod tidy` → `go mod vendor`, commit all three.

**Falak.exe** (from repo root; output only in `build/`):
```
GOOS=windows GOARCH=amd64 go build -p 1 -tags "wasm2go nodynamic" \
  -ldflags "-s -w -H windowsgui" -trimpath -o build/Falak.exe .
```
**rsrc_windows_amd64.syso**: already checked in (icon + VERSIONINFO 1.0.0.0 + manifest);
the linker picks it up automatically. To change it you would `go install
github.com/tc-hib/go-winres@latest` and regenerate from the `res/raha.rc`-equivalent
config — **not needed for 1.0.0**.
**FalakSetup-1.0.0.exe** (into `build/` only): mirror the script into
`build/nsis/installer/` (falak.nsi, raha-nsis.ico, license.txt, the built Falak.exe) with
`THIRD_PARTY_LICENSES.txt` in `build/nsis/`, run
`makensis -INPUTCHARSET UTF8 <abs path>\build\nsis\installer\falak.nsi` (OutFile two levels up
= `build/FalakSetup-1.0.0.exe`), then delete `build/nsis`.
**Falak-1.0.0-portable.zip** (into `build/` only): after building `build/Falak.exe`, run
`py installer/make_portable.py`. It zips exactly `Falak.exe` + `README.txt`
(`installer/portable/README.txt`, owner's EN+AR text — Arabic rule applies) +
`THIRD_PARTY_LICENSES.txt` with a fixed timestamp (same inputs → same zip);
`audit_paths.py` checks the script's list and, when present, the built zip.
**SHA256SUMS.txt**: `sha256sum FalakSetup-1.0.0.exe Falak-1.0.0-portable.zip > SHA256SUMS.txt`.
**github-upload-1/2.zip**: copy the repo's upload file set into `github-upload/` (docs/ included,
flat layout, **never HANDOVER.md / CLAUDE_HANDOVER.md**), zip alphabetically split into two
batches (currently 54 + 55 files).
**Gates**: `go test -count=1 ./...` · `GOOS=linux go vet -tags "wasm2go nodynamic" ./...`
(clean; `GOOS=windows` vet has 12 pre-existing unsafe.Pointer warnings — not the gate) ·
`py installer/audit_paths.py` (also forbids `RMDir /r "$INSTDIR`, `${__FILEDIR__}/` and
PowerShell). After an engine pin change: `go test -tags enginesnet -run TestEnginePins ./conv`.
**User-side**: `powershell -ExecutionPolicy Bypass -File installer\verify-install.ps1`
(silent install, shortcut targets, capture-affinity log lines, GIF with the pinned FFmpeg 9.0.2, drag CPU).
**Hard-won gotchas**: `go test | tail` swallows exit codes; gofmt realignment breaks
exact-match patches (use unique context); Python `write_text` on Windows writes CRLF — write
bytes to keep LF; `pdfium.dll` stays loaded in a test process, so its temp folder cannot be
deleted until the process exits.

## 4. Permanent rules (every task, every commit)
1. One commit per task; message style `1.0.0: <what changed>`.
2. Trailer **exactly** `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` —
   **never** a Claude-Session / generated--with line.
3. All files UTF-8 with LF line endings (vendored files stay as upstream ships them).
4. **Never change an existing Arabic string.** New Arabic only with the user's exact text,
   shown in full in your reply.
5. Version stays **1.0.0** until the user says otherwise.
6. `HANDOVER.md` / `CLAUDE_HANDOVER.md` never go into the github-upload zips.
7. Gates before every commit: `go test ./...`, `go vet` (tags), `installer/audit_paths.py`.
8. Never touch `Falak-1.0.0-release` while the Microsoft review is pending; builds go to
   `build/` only. Do not build the Setup unless the owner asks.

## 5. SOUND RULE — never modify the sound code
**Frozen verbatim at commit `4bd539b`** (old history, not in this repository; this repo's first
commit `8816700` holds the same sound files and call sites): `sound_windows.go`, `res/*.wav`,
and every call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` / `soundOn`.
No redesign, no "improvements", no volume changes — if a sound task ever comes up, **restore
from `8816700`** (`git checkout 8816700 -- sound_windows.go res/`), do not edit.

Failed attempts (all abandoned — do not retry without an explicit user request):
1. `PlaySound`-based approaches (0.31 era) — unreliable for looping/mixing.
2. WASAPI rewrite — abandoned (complexity/reliability on real hardware).
3. Custom mixer on short-lived streams — sounds died mid-wheel; replaced by the single
   persistent `waveOut` stream (commit `2923a45` "0.32 sound fix: one persistent output
   stream", after `264f9a4` "the wheel sounds never fade or die").
4. `3ddebb4` "0.32 task 1: restore the original sound code" — an earlier over-eager sound
   edit had to be rolled back the same way.
5. **confirmPCM / playConfirm** (+6 dB toggle confirmation, commit `c439372`) — broke all
   wheel sounds on real Windows; the whole sound layer was restored from `4bd539b`
   (`git checkout 4bd539b -- sound_windows.go res/` + the menu call site back to
   `playHover()`) in commit `0268ed5` "1.0.0: restore sounds from 4bd539b"; proofs: empty
   diff vs 4bd539b for sound files, identical call-site grep, gates green.

## 6. Decision history (do not relitigate)
- **WDA_EXCLUDEFROMCAPTURE removed forever** (commit `4bd539b`): it made Falak invisible
  to the user's own screen sharing/recordings and triggered NVIDIA's "preventing desktop
  capture" overlay message. Capture affinity is back to `WDA_NONE` (logged via `logAffinity`).
- **Popups are solid**: `glassBase` ignores the background — tray menu, help card and
  Compress dialog draw opaque cards in menu colors (dark {40,40,44} / light {255,255,255});
  no screen captures on open/move/drag-end; `refreshPanelGlass`/`lastBG` deleted
  (commits `d12a60c`, `4bd539b`). **Real glass lives only on the wheel.**
- **Shortcut bug root cause** (commit `8780ec3`): forward slashes in runtime paths made
  Windows mis-parse the target → "FalakFalak.exe". Backslashes everywhere; shortcuts get an
  explicit icon after `File`; `installer/audit_paths.py` guards it.
- **Greyed-out Uninstall** (commit `5cbc1bd`): the NSIS script never wrote
  `UninstallString`/`QuietUninstallString`/`InstallLocation` and DisplayIcon had forward
  slashes; fixed with the full 11-value registry block.
- **Alt-menu hover seam** (commit `7bdc834`): a hard horizontal line across the hovered top
  segment mid-animation. Root cause: `segShadow`'s `inArc` only wrapped angles upward, so the
  wheel's topmost direction (3π/2) tested as outside the arc, the shadow bbox missed the top
  of the silhouette, and the blurred shadow was cropped mid-blob — invisible under the opaque
  settled fill but showing through the translucent hover fill. Fix: normalize the angle from
  above `a1` too. Regression test: `ui/seam_test.go` (renders Alt top segment at 10–90 % of
  the hover in dark+light; fails on any row/column median step >12 levels; also covers the
  other menus' top segments). Never touched the sound code or Arabic strings while fixing it.
- **Readme/licenses** (`4ca3505`, `79112fb`): credit lines EN+AR; `THIRD_PARTY_LICENSES.txt`
  shipped in installer+portable; README points to GitHub release pages for FFmpeg (not ffmpeg.org).

Decisions from the 2026-11-08 review session (this repository's history):
- **Key helper = UAC prompt every start** (`4901549`, text `14462d5`): the scheduled task
  `Falak\KeysHelper` ran a user-writable exe with highest privileges and no prompt (silent
  elevation). No task any more; the elevated helper, the installer and the uninstaller delete
  an old one (installers fall back to one `runas` prompt). After a refused prompt the helper
  is not relaunched on every drag.
- **Engines pinned** (`9821259`): FFmpeg 9.0.2 `full_build-shared` from GyanD/codexffmpeg
  (permanent numbered releases; BtbN "latest" broke compress-to-size: `-pass 2` hit the AAC
  stream, fixed with `-pass:v`/`-passlogfile:v` in `79b7d5c`), PDFium chromium/8138,
  qpdf 12.3.2, resvg 0.49.0, each with SHA-256; a mismatch reuses the existing "download
  didn't finish" text. Never go back to "latest" links.
- **Engines folder** `%LOCALAPPDATA%\Falak\tools` (`b32152d`), not the install folder.
- **One install folder** (`6081f23`): `installDir()` = installer's InstallLocation when
  Falak.exe is there, else `%LOCALAPPDATA%\Programs\Falak`; a second copy left by old builds
  in the default folder is removed; `InstallDirRegKey` reuses the folder on update.
- **Uninstaller**: deletes only the files it installed and `RMDir` (no `/r`) on `$INSTDIR`
  (`d4b9569`); reads the keep-settings answer via `MessageBox … IDYES` (`a0456db`); no
  PowerShell — Layers cleanup is an NSIS `EnumRegValue` loop; closes Falak with
  `wmQuitUpdate` (0x8009) to `FalakMsg`, waits 5 s, then `taskkill /F`, then up to 3 s more
  for the elevated helper (`0f3acfe`).
- **Video compress** no longer holds `engineMu` during the encode (`b3a7987`).
- **Vendored dependencies** (`c311c4e`), see Section 3.

## 7. Screenshots and docs images
The README images (`docs/wheel-light-dark.png`, `docs/tray-menu.png`, `docs/compress-dialog.png`)
were **rendered headlessly in the sandbox** — no Windows UI. Because the sandbox has **no
Arabic fonts with shaping**, any Arabic text in those renders shows as "..." — **the app itself
is fine on Windows**. The **owner has replaced `docs/tray-menu.png` and
`docs/compress-dialog.png` on GitHub with real Windows screenshots** — the local copies are
outdated; do not regenerate or re-push them from the sandbox.

`docs/wheel-light-dark.png` reproduction: `cmd/themeprev` renders four wheel sets
(`ToolItems("a.mp4")`, `ToolItems("a.png")`, `FormatItems("a.mp4")`, `ToolItems("a.wav")`)
in two rows (light row then dark row) onto a sheet of `n=540` px cells (`sheet =
540×len(sets)` wide, 1080 tall), each cell over a synthetic gradient background — light
`bg(k)` = `r=245-40·fy, g=170-90·fy, b=60+30·fx` (warm), alternate cells a blue gradient
`r=40+30·fx, g=60+20·fy, b=90+60·fy` — with `ui.FrostBackground(b, 21, 1.25)` as the wheel's
`BG` and `FrostBackground(b, 4.5, 1.1)` as `Sharp`, `ui.New(1.5, …)` scale, fonts
Inter-Bold/Inter-SemiBold from `/usr/share/fonts/opentype/inter/`. The delivered image is the
light+dark pair cropped from that sheet and joined side by side (PIL, in-session). Cell size
540, wheel center (270,270).

## 8. Current state (as of 2026-11-08)
- Repository: `C:\Projects\Falak`, git `main`; history starts at `8816700` (import of the state
  after `7bdc834` "1.0.0: no seam in the hover fill mid-animation"); later commits are the
  review fixes listed in Section 6. Version still 1.0.0.
- New builds of Falak.exe / FalakSetup-1.0.0.exe exist only in `build/` (not published; their
  hashes differ from the ones under review). Manual testing on real Windows is pending.
- **GitHub repo github.com/GlitchGLH/Falak is public (MIT, author UncleGlitch).**
  The owner replaced `docs/tray-menu.png` and `docs/compress-dialog.png` with real screenshots
  on GitHub; **local copies are outdated** (leave them alone).
- **Release v1.0.0 is NOT published yet** (everything is prepared: two upload batches).
- **Microsoft submission `b8db366f-2c5d-4fba-ad8e-c4455e28841c`** (Wacatac.B!ml false
  positive) is **pending** on the files in `Falak-1.0.0-release` → never touch that folder.
- Hashes under review (copy of `SHA256SUMS.txt`):
  - `c8468accc505bc1bf82dcf36dbe0205cfb20205ac8eef494695e27a1e2063883  FalakSetup-1.0.0.exe`
  - `0bbad8511b6372a713096037f1b3da756ebc28b9173da01b61230955898e26f4  Falak-1.0.0-portable.zip`
  - (Falak.exe inside both: `b563301f172dca56b64541be5162ea52656dae22780e955f1290fa9d0b6e7200`)
- VirusTotal stays **manual** (no API key, ever).
- Real-Windows runtime checks pending (the owner runs them by hand; `installer/verify-install.ps1`
  covers part of it).

## 9. Known issues and open items
- **Compress-to-size accuracy**: a 2.00 MB target yielded 1.17 MB actual (two-pass FFmpeg
  sizing is conservative). Considered acceptable; could be tightened someday.
- **Unimplemented/gated tools**: tools only appear enabled when they apply to the dragged
  file (e.g. WHITE/BLACK BG only for alpha-capable kinds, COMPRESS not for ICO, METADATA not
  for HEIC/ICO); archive input supports EXTRACT only.
- **HEIC component license**: HEIC **input** is decoded by `github.com/gen2brain/heic`, which
  embeds the pure-Rust `heic` crate v0.1.6 compiled to WASM and transpiled to Go (`wasm2go`
  tag; CGo-free). The `nodynamic` tag turns off its optional libheif loading, so no
  libheif/libde265 (LGPL) is used. HEIC **output** goes through the downloaded FFmpeg
  (`libx265`, GPL build, fetched by the user at runtime, not shipped). The attribution sits in
  `THIRD_PARTY_LICENSES.txt` — keep it shipped in installer+portable.
- **Unsigned binaries**: SmartScreen warnings expected; the Microsoft FP review is the remedy.
- **Engine pins need manual bumps** for security fixes (edit `conv/engines.go`, run the
  `enginesnet` test). If an upstream asset ever disappears, downloads fail until re-pinned.
- **Untested on real Windows** (sandbox limits): silent install + shortcut targets, capture
  affinity log lines, GIF conversion with the pinned gyan.dev FFmpeg 9.0.2, drag CPU load,
  hover sounds on all audio devices. All covered by `installer/verify-install.ps1`.
- `gofmt -l` flags pre-existing formatting in `main_windows.go` (cosmetic only — was never
  part of the gate).
- **Deferred by the owner (2026-11-09):** portable downgrade and `wmQuitUpdate` closing Falak
  (see "Known and deferred" in the Current state section). Do not fix without the owner.

## 10. First message for the new model (copy-paste into the new session)
> You are taking over Falak 1.0.0 from a previous session. Read CLAUDE_HANDOVER.md at the
> repo root FIRST (and CLAUDE.md), then HANDOVER.md. The repo is C:\Projects\Falak (git, vendored
> dependencies). Non-negotiable: (1) never touch Falak-1.0.0-release — a Microsoft
> false-positive review on the Setup's hash
> (c8468accc505bc1bf82dcf36dbe0205cfb20205ac8eef494695e27a1e2063883) is pending; any new
> build goes to build/ only; (2) never
> modify the sound code (sound_windows.go, res/*.wav, sound call sites — frozen; same as 8816700);
> (3) never change an existing Arabic string; (4) one commit per task with only
> "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"; (5) version stays 1.0.0;
> (6) run the gates (go test ./..., go vet with tags, installer/audit_paths.py) before every
> commit. The release v1.0.0 on github.com/GlitchGLH/Falak is not published yet.
