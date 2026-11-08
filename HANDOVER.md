# HANDOVER

## Paths
- Project (this repo): `C:\Projects\Falak` (Go module `raha`, git branch `main`)
- Dependencies: vendored in `vendor/` (unmodified upstream modules; versions in `go.mod`,
  hashes in `go.sum`). A fresh clone builds offline. See BUILDING.md.

## Build Falak.exe
From the repo root, with Go >= 1.25 on PATH (output only in `build/`):

    GOOS=windows GOARCH=amd64 go build -p 1 -tags "wasm2go nodynamic" \
      -ldflags "-s -w -H windowsgui" -trimpath -o build/Falak.exe .

The checked-in `rsrc_windows_amd64.syso` (icon + VERSIONINFO 1.0.0.0 + manifest) is picked up by the linker automatically; `res/raha.rc` mirrors it for reference.

## Build FalakSetup-1.0.0.exe (NSIS 3.x)
1. Put a built `Falak.exe` into `installer/`.
2. From the repo root run (absolute script path recommended):

       makensis -INPUTCHARSET UTF8 "$(pwd)/installer/falak.nsi"

   The script resolves its assets (icon `installer/raha-nsis.ico`, license
   `installer/license.txt`, `installer/Falak.exe`) relative to its own
   directory (`${__FILEDIR__}`, with backslashes) and writes
   `FalakSetup-1.0.0.exe` two directory levels above `installer/`. To keep
   everything inside `build/`, mirror the script into `build/nsis/installer/`
   first (see CLAUDE_HANDOVER.md Section 3).

## Preview tools (cmd/, run from the repo root: `go run -p 1 ./cmd/<name>`)
- `preview` — wheel over a sample desktop → PNG files (see its source for output paths)
- `themeprev` — wheel light/dark → `/tmp/theme.png`
- `menuprev` — tray menu (EN/AR, dark/light, with logo) → `/tmp/menu.png` (arg = Inter fonts variant)
- `headerprev` — menu header sheet (EN/AR × dark/light) → `../previews/task3-header.png`
- `compressprev` — Compress dialog grid; needs fonts:
  `go run -p 1 ./cmd/compressprev -o out.png -bold /usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Bold.ttf -semi /usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Regular.ttf -debug=false`
- `helpprev`, `panelprev`, `iconprev`, `icons` — help card / panel / icon sheets
- Fonts used by the app's UI renders: Inter (Latin), Amiri for Arabic previews; DejaVu as fallback.

## What Falak writes OUTSIDE its own folder (uninstaller must remove all)
| Location | What |
|---|---|
| `HKCU\Software\Falak` | values: `Language`, `Theme`, `SoundOff`, `UseHelper`, `Welcomed`, `RelaunchedAt` |
| `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` | value `Falak` = `"<install>\Falak.exe" --startup` (legacy value `Raha` is deleted by the app) |
| `HKCU\...\Explorer\StartupApproved\Run` | value `Falak` (startup enable marker) |
| `HKCU\...\AppCompatFlags\Layers` | only ever *removed* (Run-as-admin marks on `\falak.exe`) |
| install folder: the installer's `InstallLocation` (HKCU Uninstall\Falak) when Falak.exe is there, else `%LOCALAPPDATA%\Programs\Falak` | the one `Falak.exe` (`.new`/`.old` transient); self-copy, Run value and helper all use it. A second copy that earlier builds left in the default folder is removed by the app, the installer and the uninstaller |
| `%LOCALAPPDATA%\Falak` | `falak.log` (+`.old`), crash reports |
| Scheduled task `Falak\KeysHelper` | no longer created (the helper now starts through a UAC prompt each time); a task left by earlier builds is deleted by the elevated helper, the installer and the uninstaller |
| `%LOCALAPPDATA%\Falak\tools` | downloaded engines: FFmpeg, PDFium, qpdf, resvg (pinned; each folder has `falak-engine.txt`). Earlier builds used `%LOCALAPPDATA%\Programs\Falak\tools`; the app and the uninstaller remove it |
| session-only | mutex `Local\FalakFileWheel`; reads `Themes\Personalize\AppsUseLightTheme`; no shell extensions / file associations |

## Standing rules
- One commit per task.
- Commit trailer: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` only — never a Claude-Session line.
- All files UTF-8 with LF line endings.
- Never touch `Falak-1.0.0-release` (Microsoft false-positive review pending); builds go to `build/` only.
- Never touch any Arabic string unless the exact replacement text is given; show every changed/new Arabic line in full.
- **Never modify the sound code.** `sound_windows.go`, `res/*.wav` and every call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` / `soundOn` are frozen verbatim at commit `4bd539b` (the tested-good build; same files as this repo's first commit `8816700`). No redesign, no "improvements", no volume changes; if a sound task ever comes up again, restore from `8816700` instead of editing.

## State on 2026-11-07 (before the 2026-11-08 review; see CLAUDE_HANDOVER.md Sections 6 and 8 for what changed since)
- Final head: `1.0.0: restore sounds from 4bd539b` — the sound code (engine, `res/`, and every call site, incl. the menu sound toggle playing `playHover()` at its old level) is byte-identical to `4bd539b`; `confirmPCM` / `playConfirm` are gone.
- Kept from this session, untouched: compress target size fix, tray menu animation, GIF 10 s cap + progress, solid popups, licenses file, shortcut/capture fixes.
- Version stays 1.0.0; `FalakSetup-1.0.0.exe`, the portable zip, `SHA256SUMS.txt` and the two GitHub-upload batches are rebuilt from this head.
- User-side verification only: `installer/verify-install.ps1` on real Windows; VirusTotal stays manual (no API key).
