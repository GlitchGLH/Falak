# CLAUDE_HANDOVER — Falak 1.0.0

**READ THIS FIRST (new session):** You are taking over Falak, a finished, released-quality
Windows drag-and-drop file converter, at version **1.0.0**. Two things are in flight and
must not be disturbed: a **Microsoft antivirus false-positive review** and a **not-yet-published
GitHub release**. **Do not rebuild any binary** — the shipped hashes are under review.
Everything else (code review, fixes, new tasks, screenshots, docs) proceeds normally
under the rules at the bottom of this file.

---

## 1. What Falak is

Falak (فلك — "orbit/celestial sphere", internal Go module name `raha`) is a **drag-and-drop
conversion wheel for Windows 10/11 (x64)**. Drag any file with **Shift held**, and a frosted-glass
radial wheel pops up around the cursor. Each slice is one target format or tool; drop on a slice
to convert. It is fully bilingual (**English / Arabic with RTL mirroring**), light/dark aware,
renders its whole UI itself (no system controls), and converts with bundled/open engines
(FFmpeg, PDFium, qpdf, resvg, libavif/aom/dav1d/libwebp, a pure-Rust HEIC crate) downloaded
on demand to `%TEMP%\falak-tools`. Author: **UncleGlitch**; license **MIT**;
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
- **Key-reading helper**: opt-in elevated helper (scheduled task `Falak\KeysHelper` via
  `schtasks`) so key state can be read while an elevated window has focus.

### Conversions per format (see README.md's table for the authoritative matrix)
- **Images**: between JPG, PNG, WEBP, GIF, AVIF, TIFF, BMP, ICO, HEIC (input), plus **SVG**
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

## 2. Full file map (110 tracked files)

**Root docs/config**: `README.md` (EN/AR product page, format table, credit lines) ·
`BUILDING.md` (build steps) · `HANDOVER.md` (short internal handover: paths, commands, rules) ·
`CLAUDE_HANDOVER.md` (this file) · `LICENSE` (MIT) · `RELEASE_NOTES.md` ·
`THIRD_PARTY_LICENSES.txt` (26 Go modules + libavif/aom/dav1d/libwebp texts; ships in installer
+ portable) · `THIRD_PARTY_NOTICES.md` · `go.mod` (module `raha`; `replace` → `../deps/...`)
· `.gitignore` (ignores `*.exe`, incl. `installer/Falak.exe`) · `rsrc_windows_amd64.syso`
(checked-in icon+VERSIONINFO 1.0.0.0+manifest, linked automatically).

**App entry/Windows layer**: `main_windows.go` (App lifecycle, drag sessions, tick/present,
capture-free popups, tray init, `logAffinity`) · `droptarget_windows.go` (COM drag&drop
target) · `tray_windows.go` (tray icon) · `menu_windows.go` (tray menu window, rows, actions,
fade/slide animation via `presentLayeredA`) · `compdialog_windows.go` (Compress dialog host,
no captures) · `help_windows.go` (How-to-use host) · `win32_windows.go` (Win32 decls) ·
`input_windows.go` (keyboard state) · `elevation_windows.go` (run-as-admin) ·
`helper_windows.go` (KeysHelper opt-in) · `install_windows.go` (self-install, registry) ·
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
compress-to-size two-pass) · `gif_test.go` · `target_test.go` · `heif.go` (HEIC via pure-Rust
crate cgo) · `imgdocx.go` (JPG/PNG→DOCX) · `office.go`+`office_windows.go` (DOCX) · `pdf.go` ·
`pdfium_windows.go`+`pdfium_other.go` (PDFium) · `qpdf.go` · `svg.go` (resvg) · `raw.go` ·
`merge.go` · `metadata.go` · `sevenzip_write.go` · `text.go` · `textarchive.go` ·
`wic_windows.go` (WIC codecs) · `lang.go` · `progress.go`(+test) · `tempfiles.go` ·
`conv_test.go`/`media_test.go`/`text_test.go`.

**`cmd/`** (dev-only preview/bench tools): `preview`, `themeprev`, `menuprev`, `headerprev`,
`compressprev`, `helpprev`, `panelprev`, `hoverprev`, `entrance`, `toolprev`, `iconprev`,
`icons`, `pillcrop`, `opacity`, `bench`, `canceltest`, `ptime`, `qtest`, `t`, `pdfdl`,
`pdftest`, `pdftools`, `wictest`.

**`installer/`**: `falak.nsi` (NSIS; backslash runtime paths; full registry block; desktop
+ Start Menu shortcuts with explicit icon; ships THIRD_PARTY_LICENSES.txt; `BrandingText
"Falak 1.0.0 · By UncleGlitch"`; OutFile two levels above `installer/`) · `license.txt` ·
`raha.ico`/`raha-nsis.ico` · `audit_paths.py` (static NSIS path audit — part of the gate) ·
`verify-install.ps1` (real-Windows runtime verification) · `Falak.exe` (gitignored build drop).

**`res/`** (**FROZEN sounds** `hover.wav`/`appear.wav`/`drop.wav`; `raha.ico`, `raha.rc`,
`raha.manifest`, `icon-1024.png`).

**`docs/`**: `wheel-light-dark.png`, `tray-menu.png`, `compress-dialog.png` — see Section 6.

## 3. Build environment (exact) and commands

| Tool | Version |
|---|---|
| Go | **go1.27.3** linux/amd64 (needs ≥ 1.24) |
| NSIS (makensis) | **v3.09-4** |
| Python | **3.12.3** (PIL/Pillow installed; used for audits + docs composition) |
| go-winres | **not installed in the sandbox** — the `.syso` is prebuilt and checked in |

Layout: repo at `~/workspace/falak-build/falak-src`, deps at `~/workspace/falak-build/deps`
(**must sit next to the repo** — `go.mod` has `replace … ../deps/...`; pins like
`gen2brain/avif|heic|webp v0.0.0`, `srwiley/oksvg|rasterx v0.0.0` do not exist upstream —
they are wasm2go builds generated for this project; **never** drop the `replace` lines or
try `GOPROXY` for them).

**Falak.exe** (from repo root):
```
GOFLAGS=-mod=mod GOPROXY=off GOSUMDB=off GOMEMLIMIT=2500MiB \
GOOS=windows GOARCH=amd64 go build -p 1 -tags "wasm2go nodynamic" \
  -ldflags "-s -w -H windowsgui" -trimpath -o installer/Falak.exe .
```
**rsrc_windows_amd64.syso**: already checked in (icon + VERSIONINFO 1.0.0.0 + manifest);
the linker picks it up automatically. To change it you would `go install
github.com/tc-hib/go-winres@latest` and regenerate from the `res/raha.rc`-equivalent
config — **not needed for 1.0.0**.
**FalakSetup-1.0.0.exe**: put the built `Falak.exe` into `installer/`, then
`makensis -INPUTCHARSET UTF8 "$(pwd)/installer/falak.nsi"` (absolute script path; output
lands two directories above `installer/`, i.e. `~/workspace/falak-build/`).
**Falak-1.0.0-portable.zip**: zip exactly `Falak.exe` + `README.txt` + `THIRD_PARTY_LICENSES.txt`
(README.txt is the short portable read-me, regenerated in-session; content in the delivered zip).
**SHA256SUMS.txt**: `sha256sum FalakSetup-1.0.0.exe Falak-1.0.0-portable.zip > SHA256SUMS.txt`.
**github-upload-1/2.zip**: copy the repo's upload file set into `github-upload/` (docs/ included,
flat layout, **never HANDOVER.md / CLAUDE_HANDOVER.md**), zip alphabetically split into two
batches (currently 54 + 55 files).
**Gates**: `go test ./...` · `go vet -tags "wasm2go nodynamic" ./...` (host vet + windows
build + tests; `GOOS=windows go vet` has pre-existing unsafe.Pointer warnings — not the gate)
· `python3 installer/audit_paths.py`.
**User-side**: `powershell -ExecutionPolicy Bypass -File installer\verify-install.ps1`
(silent install, shortcut targets, capture-affinity log lines, GIF vs BtbN ffmpeg, drag CPU).
**Hard-won gotchas**: `set -e` unreliable in the sandbox shell (use explicit `&&`); `go test | tail`
swallows exit codes; gofmt realignment breaks exact-match patches (use unique context);
wine is broken and 32-bit Windows code cannot execute in the sandbox at all.

## 4. Permanent rules (every task, every commit)
1. One commit per task; message style `1.0.0: <what changed>`.
2. Trailer **exactly** `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` —
   **never** a Claude-Session / generated--with line.
3. All files UTF-8.
4. **Never change an existing Arabic string.** New Arabic only with the user's exact text,
   shown in full in your reply.
5. Version stays **1.0.0** until the user says otherwise.
6. `HANDOVER.md` / `CLAUDE_HANDOVER.md` never go into the github-upload zips.
7. Gates before every commit: `go test ./...`, `go vet` (tags), `installer/audit_paths.py`.

## 5. SOUND RULE — never modify the sound code
**Frozen verbatim at commit `4bd539b`**: `sound_windows.go`, `res/*.wav`, and every call
site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` / `soundOn`. No redesign,
no "improvements", no volume changes — if a sound task ever comes up, **restore from
`4bd539b`**, do not edit.

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
- Final commit: `7bdc834` "1.0.0: no seam in the hover fill mid-animation"; tree clean.
- **GitHub repo github.com/GlitchGLH/Falak is public (MIT, author UncleGlitch).**
  The owner replaced `docs/tray-menu.png` and `docs/compress-dialog.png` with real screenshots
  on GitHub; **local copies are outdated** (leave them alone).
- **Release v1.0.0 is NOT published yet** (everything is prepared: two upload batches).
- **Microsoft submission `b8db366f-2c5d-4fba-ad8e-c4455e28841c`** (Wacatac.B!ml false
  positive) is **pending** → **the binaries must stay byte-identical; do NOT rebuild**.
- Final hashes (copy of `SHA256SUMS.txt`):
  - `c8468accc505bc1bf82dcf36dbe0205cfb20205ac8eef494695e27a1e2063883  FalakSetup-1.0.0.exe`
  - `0bbad8511b6372a713096037f1b3da756ebc28b9173da01b61230955898e26f4  Falak-1.0.0-portable.zip`
  - (Falak.exe inside both: `b563301f172dca56b64541be5162ea52656dae22780e955f1290fa9d0b6e7200`)
- VirusTotal stays **manual** (no API key, ever).
- Real-Windows runtime checks pending via `installer/verify-install.ps1` (sandbox cannot run
  Windows code; wine is broken).

## 9. Known issues and open items
- **Compress-to-size accuracy**: a 2.00 MB target yielded 1.17 MB actual (two-pass FFmpeg
  sizing is conservative). Considered acceptable; could be tightened someday.
- **Unimplemented/gated tools**: tools only appear enabled when they apply to the dragged
  file (e.g. WHITE/BLACK BG only for alpha-capable kinds, COMPRESS not for ICO, METADATA not
  for HEIC/ICO); archive input supports EXTRACT only.
- **HEIC component license**: the HEIC decode path uses a pure-Rust crate (no LGPL); the
  attribution sits in `THIRD_PARTY_LICENSES.txt` — keep it shipped in installer+portable.
- **Unsigned binaries**: SmartScreen warnings expected; the Microsoft FP review is the remedy.
- **Untested on real Windows** (sandbox limits): silent install + shortcut targets, capture
  affinity log lines, GIF conversion against the user's BtbN ffmpeg build, drag CPU load,
  hover sounds on all audio devices. All covered by `installer/verify-install.ps1`.
- **Sandbox quirks to remember**: no Arabic fonts (renders show "..."), wine broken,
  `gofmt -l` flags pre-existing formatting in `main_windows.go` (cosmetic only — was never
  part of the gate).

## 10. First message for the new model (copy-paste into the new session)
> You are taking over Falak 1.0.0 from a previous session. Read CLAUDE_HANDOVER.md at the
> repo root FIRST, then HANDOVER.md. The repo is at falak-src/ with deps/ next to it.
> Non-negotiable: (1) do NOT rebuild FalakSetup-1.0.0.exe, Falak.exe, the portable zip or
> SHA256SUMS.txt — a Microsoft false-positive review on the Setup's hash
> (c8468accc505bc1bf82dcf36dbe0205cfb20205ac8eef494695e27a1e2063883) is pending; (2) never
> modify the sound code (sound_windows.go, res/*.wav, sound call sites — frozen at 4bd539b);
> (3) never change an existing Arabic string; (4) one commit per task with only
> "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"; (5) version stays 1.0.0;
> (6) run the gates (go test ./..., go vet with tags, installer/audit_paths.py) before every
> commit. The release v1.0.0 on github.com/GlitchGLH/Falak is not published yet.
