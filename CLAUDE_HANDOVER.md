# CLAUDE_HANDOVER — Falak 1.0.0

## Current state (latest session)

Written at the end of the 2026-11-09 session (after the 2026-11-08/09 review session). This
section wins over anything older below it. Arabic is quoted character for character.

### Where things are
- Repository: `C:\Projects\Falak`, git branch `main`, local config `core.autocrlf=false`,
  `user.name=UncleGlitch`, `user.email=135931273+GlitchGLH@users.noreply.github.com` (local
  only; the owner's choice, 2026-11-09). History starts at `e29e934` (import of the handed-over
  state).
- **GitHub (2026-11-09): release v1.0.0 PUBLISHED**:
  https://github.com/GlitchGLH/Falak/releases/tag/v1.0.0 (title "Falak 1.0.0", Latest, not
  draft / prerelease; assets `FalakSetup-1.0.0.exe`, `Falak-1.0.0-portable.zip`,
  `SHA256SUMS.txt`, downloaded back and checked with `sha256sum -c`, the zip's Falak.exe too).
  Remote `origin` = https://github.com/GlitchGLH/Falak.git; default branch `main` = this local
  history (pushed with `--force-with-lease`, first at `269428f`). The old GitHub `main`
  (`c47bb7a` "Remove duplicate screenshot", with the owner's real screenshots) is kept in the
  remote branch **`old-upload`** - do not delete it. Annotated tag `v1.0.0` -> `57f35db` (the
  build commit). Push with the GitHub CLI login (`gh auth setup-git`, account GlitchGLH).
  Never push `backup-before-email` or `build/`; never force-push `main` again.
- **Email rewrite (2026-11-09):** every commit from the import to `bcd19cc` was rewritten with
  `git filter-branch --env-filter` to author and committer
  `UncleGlitch <135931273+GlitchGLH@users.noreply.github.com>`. Trees, messages, names and
  dates are unchanged (`git diff backup-before-email bcd19cc` is empty; every commit pair has
  the same tree). The old history is kept in the local branch `backup-before-email` (never push
  it). Commit messages written before the rewrite still quote the old hashes; map them with
  this table (old -> new):

  | Old | New |
  |---|---|
  | `8816700` | `e29e934` |
  | `a0456db` | `30476e0` |
  | `d4b9569` | `fa55486` |
  | `b3a7987` | `8d56baf` |
  | `7565407` | `7a081a1` |
  | `79b7d5c` | `fc5e411` |
  | `3db401c` | `657d478` |
  | `f6eedbc` | `ae9fbcb` |
  | `4901549` | `7aed3ae` |
  | `cf15b37` | `256b8da` |
  | `0f3acfe` | `b8bbda8` |
  | `14462d5` | `8bd429d` |
  | `9821259` | `3c96dbb` |
  | `b32152d` | `fae66c4` |
  | `6081f23` | `c5be6f9` |
  | `c311c4e` | `af994d0` |
  | `bd93a63` | `8096641` |
  | `50cdd9d` | `58c30f9` |
  | `48618a9` | `04bb412` |
  | `3967648` | `b759137` |
  | `d8c6433` | `50e2895` |
  | `1dbcfbd` | `b57ba4f` |
  | `4fc1ea2` | `1df38b2` |
  | `7516d74` | `8439bc2` |
  | `bc37f7d` | `4f45c70` |
  | `8ea1363` | `d122770` |
  | `a52b66c` | `405e2a7` |
  | `1927fab` | `b98e64c` |
  | `0b40241` | `26f752d` |
  | `06b34cd` | `53fee02` |
  | `7e0b4eb` | `37d6258` |
  | `ed09f1c` | `312d695` |
  | `87d522e` | `aecd928` |
  | `fd5b793` | `c504236` |
  | `6618323` | `3bdb5b0` |
  | `788ec66` | `7d67d54` |
  | `b178c76` | `bcd19cc` |
- Working tree clean after this handover commit.
- Version: **1.0.1** everywhere (bump `961a735`, owner's decision 2026-11-09). The published
  v1.0.0 release and its assets stay untouched forever.
- Release folders (Microsoft false-positive review pending on their hashes): never modify;
  read only with the owner's explicit permission:
  - `C:\Users\Glitch\Desktop\Falak-1.0.0-release`
  - `C:\Users\Glitch\Desktop\Falak with Claude Code\Falak-1.0.0-release`

### New features, phase 1 (2026-11-15; version stays 1.0.1)
- `2c8bf1c` "1.0.1: compressing several files saves them in a dated folder": when two or more
  files are compressed together (Compress tool of any kind: `compress`, `v:compress`,
  `a:compress`, `p:compress`; counted after `applicable()`), each source folder gets a new
  folder `Compressed 2026-11-15 05-30` (local time at the drop; `conv.CompressFolderName`) and
  that folder's copies go in it (phase 2: under the ORIGINAL name, see below). Owner's decision:
  files from different folders get one dated folder in EACH source folder. An existing folder
  is never reused (`makeFolder` = `os.Mkdir` on the `nameSeries`, so " (1)", " (2)" ...); a
  folder left empty (all files failed or the job was cancelled) is removed at the end; a folder
  that cannot be created leaves that folder's copies beside the originals. One file is
  unchanged. The output folder reaches the tools through the context (`conv.WithOutDir`,
  `outPath`). The compress dialog is untouched (it only opens for ONE video; several videos
  are compressed Balanced without it, as before). Code: `conv/compressfolder.go`, tests
  `conv/compressfolder_test.go`; job wiring in `main_windows.go` (`onDrop`).
- `4f166b1` "1.0.1: image-only archives convert to PDF": ZIP, RAR and 7Z show a `PDF` target
  (always shown; enabled only when every file is an image Falak reads, at least one).
  Companion files are IGNORED (owner's decision): `Thumbs.db`, `desktop.ini`, `.DS_Store` and
  anything under a `__MACOSX` folder. Pages: one image per page via `imagesToPDF` (the MERGE
  PDF writer; JPEG kept as is), in natural order folder by folder along the path (2 before 10,
  case ignored, extension only breaks ties: `img` before `img 2`; `sortPages`). The images are
  extracted to a hidden temp folder as `<page><ext>` (no archive name used as a path), the PDF
  is `<archive name>.pdf` beside the archive (never replaces). Reading the list is fast (ZIP
  586 MB / 2000 entries: 10 ms; 7Z 2000 entries: under 8 ms; RAR skips the data by seeking)
  and is cached per path+size+time (the wheel asks every frame). An SVG inside downloads resvg
  first (`NeedsResvg(path, id)`). TAR/GZ do not offer PDF. Two errors that the wheel cannot
  reach got the owner's Arabic in phase 2 ("This archive doesn't hold only images", "Couldn't
  read %s from the archive"). No RAR test (no WinRAR on this machine to
  make one). Code: `conv/archivepdf.go`, tests `conv/archivepdf_test.go`.
- Test build (not a release): `build/phase1/Falak.exe` from `4f166b1`
  (`vcs.modified=false`), 38,734,848 bytes, SHA-256
  `63a3eddc5357de3d205f24f8eba1ad9dcec0e09ef3f7ee52b441c4a33b68f3c5`. Kept apart so the 1.0.1
  files and `build/SHA256SUMS.txt` stay valid. Running it replaces the installed Falak.exe (the
  self-copy); FalakSetup-1.0.1.exe puts 1.0.1 back.
- Video windows (Split, Snapshots, Speed, Crop, Trim, like the Tangerine app): investigated
  only, nothing implemented. Proposed order: Trim + Snapshots, then Split + Speed, then Crop.
  Every English string they need waits for the owner's Arabic.

### New features, phase 2 (2026-11-15; version stays 1.0.1)
Owner's decisions: the video windows REPLACE the one-click SPEED 2x, SNAPSHOT and SPLIT 1/2
(SNAPSHOT replaced now; SPEED and SPLIT stay until their windows are built); the preview is
silent (Falak has no audio path besides the untouchable sound code; output files keep their
sound); Trim and Split cut exactly by re-encoding, never at the nearest key frame; README is
updated once, before publishing.
- `50c9d8b` "1.0.1: compressed copies in the dated folder keep the original name": in a
  `Compressed <date> <time>` folder each copy has the original name (`photo.jpg`); only the
  extension follows the output format (`clip.mov` -> `clip.mp4`, `scan.bmp` -> `scan.jpg`).
  Same name twice -> " (n)" series. One file: unchanged (`-compressed` beside the original).
- `4ef320e` "1.0.1: Arabic text for the archive-to-PDF errors": owner's Arabic for the two
  phase 1 messages.
- `0b6118d` "1.0.1: Trim Video window" (the shared video base is in this commit too):
  - Shared base, reusable by Split, Speed and Crop: `conv/videoframes.go` (`ProbeVideo`,
    `VideoInfo.FrameSeek` = a quarter frame early so FFmpeg's exact seek lands ON frame k,
    `FrameAt` = one FFmpeg run per picture, BMP over a pipe, scaled to fit; exact or key
    frame; `Thumbs` = 3 at a time; `OpensWindow(id)`), `ui/videobase.go` (the Compress
    dialog's solid card, X top left, title/file name, preview area, thumbnail strip, button,
    `timeX`/`shareAt` that mirror time in Arabic, `FormatClock` = 0:04.18, cut not rounded),
    and the window manager `videowin_windows.go` (one window at a time; a worker goroutine
    hands frames over under `vwin.mu` + `wmVideoUpdate`, latest frame request wins, results
    of a closed window are dropped by `gen`). FFmpeg missing: the window shows "Preparing the
    video..." with a thin bar while the same `EnsureFFmpeg` download as every video job runs
    under `engineMu`. Measured: an exact 1080p H.264 frame 0.15-0.2 s, a thumbnail 0.14 s.
  - Trim: `ui/trimdlg.go`; wheel item `v:trim` "TRIM" (icon "trim") after COMPRESS. Two
    handles (one frame apart at least), the parts left out dimmed, the preview shows the
    frame under the dragged handle (the end handle: the last frame kept), "%s selected",
    Trim / Enter starts the job "Trimming" in the progress window; Esc / X close; the window
    can be dragged by its empty parts. Output `<name>-trimmed.<ext>` (`.mp4/.mov/.mkv` kept,
    else `.mp4`), x264 CRF 16 + AAC 192k, exact cut (`conv.TrimVideo`, `-ss` before `-i`).
    Under 0.1 s: "The selection is too short". Several videos dragged: TRIM is disabled
    (phase 2 decisions, below). The compress job launcher became `launchVideoJob` (shared, no copy of
    its texts).
- `628ff8a` "1.0.1: Video Snapshots window replaces the one-click snapshot": `ui/snapdlg.go`,
  `conv/snapshot.go`. Preview, scrubber, previous/next frame buttons (and Left/Right keys,
  towards that side), "current / total" time; Save Snapshot / Enter saves the frame shown
  (`SnapshotAt` at the same seek: the saved PNG is pixel-identical to the preview's source
  frame, tested) as `<name>-snapshot.png`, then `<name>-snapshot (1).png`, `(2)`...; the
  window stays open; "Saved" shows for 2 s. The old one-click case is gone from `runVideo`.
- Arabic layout: the timeline, the strip, the handles, the scrubber and the bar run from the
  right; "previous" sits on the right (its triangle points right) because earlier time is on
  the right. «المدة المحددة: 0:05.41» reads label first, the time stays in order. The
  "current / total" line is drawn "total / current" in Arabic so the current time is on the
  right where reading starts. Screenshots (local only, from `shots_windows_test.go`
  `TestShotsPhase2`): `C:\Users\Glitch\Desktop\Falak phase 2 shots`.
- Test build (not a release): `build/phase2/Falak.exe` from `628ff8a`
  (`vcs.modified=false`), 38,825,472 bytes, SHA-256
  `a2a1bcabf9522e350da33311d423f86c20068b4ad9a078ea5ef175a2217853f5`. `build/Falak.exe`,
  `build/phase1` and `build/SHA256SUMS.txt` untouched. The windows were not run on a real
  desktop by Claude (running a build replaces the installed Falak); the owner tests them.

### Phase 2 decisions (owner, 2026-11-15)
- Frame buttons in Arabic: KEEP the mirrored layout ("previous frame" on the right, its
  arrow pointing right). No code change.
- `a8957be` "1.0.1: trim and snapshot are disabled for several videos": `videoTools(paths)`
  marks TRIM and SNAPSHOT disabled (shown, never hidden) when more than one file is dragged;
  the other video tools stay enabled. Test `TestWindowToolsOneVideo` (conv).
- `7f627c3` "1.0.1: specific message when a snapshot can't be saved": a failed save shows
  "Couldn't save the snapshot" «تعذّر حفظ اللقطة» (owner's text) as the title, with the file
  name and the error below, in a Falak message box plus the balloon (`showError(title,
  text)`; `showFailure` = `showError("Conversion failed", ...)`).
- Test build (not a release): `build/phase2b/Falak.exe` from `7f627c3`
  (`vcs.modified=false`), 38,825,472 bytes, SHA-256
  `44f2f0bb63ae3fb8eb5087663bb3a1c8fa941bbdd4ea0306b1eff4686a0a8c26`. Contains phases 1 and
  2 and these decisions. `build/Falak.exe`, `build/phase1`, `build/phase2` and
  `build/SHA256SUMS.txt` untouched (verified).

### Phase 2 test results and fixes (owner's test of `build/phase2b`, 2026-11-15)
- PASSED: archive to PDF (ZIP, 7Z made by Falak, page order 1, 2, 10, " (n)" names, PDF
  disabled for a mixed ZIP), Trim window, Snapshots window (names, same frame as the
  preview), TRIM/SNAPSHOT disabled for two videos, "Preparing the video…" without FFmpeg,
  Arabic Trim and Snapshots windows (texts, mirroring, time display).
- FAILED, fixed in `701e0af` "1.0.1: compressing several files really uses the dated
  folder": 2 JPGs + a MOV + a PDF dropped on COMPRESS gave no folder, only the first file's
  kind was compressed (old `-compressed` name), the JPGs got nothing and no error showed.
  Cause (falak.log: "dropped 4 file(s) on p:compress", then "skipped ...: p:compress does
  not apply to it" for the other three): Compress has one ID per kind (`compress`,
  `v:compress`, `a:compress`, `p:compress`) and the wheel shows the FIRST file's; the drop
  kept only files whose own items had that exact ID (`applicable`), so one file was left,
  `len(paths) >= 2` was false and no folder was made; skipped files were only logged.
  Fix: `planDrop` gives every file its own kind's Compress (`conv.CompressIDFor`); a file
  that can't be compressed is listed as "<name>: This option doesn't apply to the selected
  files" in the failure window; the job body is `runDropFiles` (dated folder when 2+ files).
  Gap in the tests: phase 1 tests called `conv.Run` with `WithOutDir` on files of ONE kind
  and never went through the drop path (`applicable` + the wheel's one ID). New test
  `TestDropCompressMixed` (drop_windows_test.go) runs the real case through
  `planDrop`/`runDropFiles`; it failed on the old logic with exactly the owner's result.
- ISSUE, fixed in `4d02f68` "1.0.1: error messages show the real cause, not FFmpeg's
  progress line": a snapshot into a folder in C:\Program Files showed "Conversion failed:
  frame= 1 fps=0.0 ... speed=1.66x". Cause: `ffmpegError` took FFmpeg's LAST stderr line,
  the statistics line; the real reason was that `.falak-tmp` could not be created (no
  permission), so FFmpeg only said "No such file or directory". Fix: `checkTempDir` before
  every FFmpeg run returns "<folder>: <Windows' reason>" (e.g. "Access is denied."), and
  `ffmpegError` now picks the FIRST line stating an error (skipping statistics, summaries,
  image2's pattern warning and the follow-up lines). All FFmpeg paths share it (convert,
  compress, two-pass, GIF palette, HEIC). The snapshot window shows `conv.Cause(err)`
  (without "Conversion failed:"); the job failure window keeps "Conversion failed: <cause>".
  Tests: `TestFFmpegErrorLine` (the owner's stderr), `TestSnapshotUnwritableFolder`.
- `bc9f65e` "1.0.1: popups open and close as smoothly as the wheel": `ui/popanim.go`
  (`PopOpen`, `PopOpenDone`, `PopClose`, `PopFrame`) holds the wheel's numbers (open from
  40%, exp 35 ms, alpha 50% -> 100% by the third frame; close: fade + shrink 8%, ease-out,
  160 ms); the wheel itself is unchanged. `popOpenLayered` / `animateLayeredClose`
  (menu_windows.go) apply them to the tray menu (its old slide/fade is gone), How to use,
  Compress dialog, Trim and Snapshots windows; the progress window already popped in on
  the same curve (`ui/panel.go` now calls `PopOpen`) and now also closes with it on X, on
  finishing and when "Done" ends (`closePanelAnimated`). No focus change: each window is
  shown exactly as before. `ui/zoom.go` (ZoomFrame) removed. Not animated: Windows message
  boxes (`messageBox`: errors, helper hint, "already running", drag-and-drop error) and
  the progress window when a new drag sends its job to the background (the wheel must
  appear at once); both waiting for the owner's decision.
- Test build: `build/phase2c/Falak.exe` from `bc9f65e` (`vcs.modified=false`), 38,836,736
  bytes, SHA-256 `30c05f5d5a3bba0da06be4cff34ab61457f308490322e7af15a95adce6a3e7ac`.
  Other build folders and `build/SHA256SUMS.txt` untouched (verified).

### Falak message cards (owner's decisions 2026-11-15; `build/phase2c` not tested yet)
- Decision 1: replace EVERY Windows message box in Falak.exe with Falak's own card (solid
  card, the wheel's open/close). Not the installer/uninstaller (NSIS) messages.
- Decision 2: the progress window keeps hiding INSTANTLY when a new drag sends its job to
  the background (no delay for the wheel). Do not animate it.
- Approved Arabic for the question's buttons (owner, 2026-11-15): Yes = «نعم», No = «لا».
- `3aef544` "1.0.1: Falak cards replace the Windows message boxes": `ui/msgcard.go`
  (`MessageCard`: the Compress dialog's solid card via `glassBase`, X top left, an
  info/error/question icon and the title (`appName`), the text wrapped (a too-long path is
  cut; more than 14 lines scroll with the wheel), buttons along the bottom, the first in the
  accent color, Arabic mirrors title, text and button order) and `msgcard_windows.go`
  (`showMessageCard`, `messageBox`). Like MessageBox it blocks and runs its own message
  loop (the calling thread's windows keep working; WM_QUIT is passed on), works from any
  goroutine (window on that locked thread) and in the second process; its own `ui.Fonts`
  (faces are not thread-safe); topmost, on the taskbar (no WS_EX_TOOLWINDOW, the exe's
  icon), shown then `SetForegroundWindow` as before. Keys: Enter/Space press the focused
  button (the first at start), Esc / X / Alt+F4 = the cancel result (OK cards: closed;
  the helper question: No), Tab / Shift+Tab / arrows move between Yes and No (mirrored in
  Arabic), Ctrl+C copies the title and text (as MessageBox did). Drag the card from
  anywhere but its controls. The five replaced: the already-running note (main, second
  process, then exits), the drag-and-drop registration error (then exits), every error of
  `showError` (job failures, snapshot/trim errors), the Key-reading helper hint, and the
  helper question (Yes/No, `offerHelper`, Yes starts the helper). `pMessageBoxW`, `rtl()`
  and the unused MB_ constants removed; `appFontPaths()` shared with main. `ui.Visual`
  now mirrors « » in Arabic runs (they came out reversed in the cards: «فلك»).
  Tests: `TestMessageCardButtons`, `TestMessageCardMirror`, `TestMessageCardWrap`,
  `TestVisualGuillemets` (ui), `TestNextFocus`. A one-off live run (not committed) showed
  the window and closed it with Enter, Esc, Tab+Enter, Right+Enter and WM_CLOSE.
- Test build: `build/phase2d/Falak.exe` from `3aef544` (`vcs.modified=false`), 38,871,552
  bytes, SHA-256 `fcb46d381f7882e0cdcd24538fac619918feb319122d49092d30847aa6748e4a`.
  Other build folders and `build/SHA256SUMS.txt` untouched (verified).

### Phase 2d test results (owner, 2026-11-15)
- EVERYTHING PASSED on `build/phase2d`: multi-file compress (dated folder, original
  names), the .txt error card, the real cause "Access is denied" (Ctrl+C copies it), the
  "already running" card, the helper question and hint cards in Arabic (correct « »,
  «نعم» on the right), the open/close animation of every popup, the progress window
  hiding instantly on a new drag.
- Decision: the helper question card KEEPS X and Esc (both mean "No").

### New features, phase 3 (2026-11-15; version stays 1.0.1)
- Approved Arabic (owner, 2026-11-15): "Split Video" «تقسيم الفيديو», "Change Video Speed"
  «تغيير سرعة الفيديو», "Add Split" «إضافة نقطة تقسيم», "Split" «تقسيم», "Change Speed"
  «تغيير السرعة», "Speed" «السرعة», "Splitting" «جار التقسيم» (was already the Split title),
  "Changing the speed to %s" «جار تغيير السرعة إلى %s», "Add at least one split point"
  «أضف نقطة تقسيم واحدة على الأقل», counter: 0 «لا توجد نقاط تقسيم», 1 «نقطة تقسيم واحدة»,
  2 «نقطتا تقسيم», 3-10 «%d نقاط تقسيم», 11+ «%d نقطة تقسيم» (English "0 split points",
  "1 split point", "%d split points").
- `f0e155a` "1.0.1: tests use only approved Arabic text": the message card tests use
  «طريقة الاستخدام» and the helper hint's own sentence instead of two unapproved phrases.
  RULE: test files may only contain the app's approved Arabic too.
- `ad3d128` "1.0.1: file names read first in Arabic message lines". The owner reported the
  card line "notes.txt: هذا الخيار …" with the name on the LEFT. It was NOT: a pixel match of
  the owner's own screenshot (`Pictures\Screenshots\Screenshot 2026-11-15 170624.png`)
  against the off-screen render gave 0 differing pixels, and its far right reads
  ":notes.txt". The owner then confirmed the card is correct. Fixed instead: the Arabic
  progress window drew "name · 45%" with the percent on the RIGHT (read first); now
  `ui.subLine` puts the name first on the right and "45% ·" on its left. Tests:
  `TestSubLineArabic`, `TestCardFileNameFirstArabic` (pixel test: the right end of the
  Arabic line is the file name). Before/after PNGs in `Desktop\Falak phase 3 shots\task2-*`.
- `cf39e6f` "1.0.1: Split Video window replaces split in half": `ui/splitdlg.go`
  (preview, 8-thumbnail strip, white playhead, accent split points each with a round ×
  above it; "Add Split" (quiet button, dimmed when no point fits at the playhead) on the
  reading-start side, the counter on the other; red error line; Split), host in
  `videowin_windows.go` (`splitPlayhead`, `splitSnap`, `splitCountText`, `splitStart`),
  `conv/videotools.go` `SplitVideo` + `SplitGap`. A point is REMOVED by clicking its ×
  (no text needed) or with Delete (the point last added or dragged). Points and the
  playhead sit on frame starts; points keep `SplitGap` (whole frames, at least 0.1 s) from
  the ends and each other; dragging a point moves the preview with it. Click/drag on the
  strip moves the playhead; Left/Right step one frame (mirrored in Arabic); Enter = Split.
  The playhead time shows in the preview's corner (reading-start side). Split with no
  point shows «أضف نقطة تقسيم واحدة على الأقل» and stays open. Job: ONE FFmpeg run, x264
  CRF 16 with `-force_key_frames` at the cuts and the segment muxer at the same times
  (cut = FrameSeek of the point's frame, a quarter frame early, so the key frame lands on
  that very frame), into "<name> - parts" (new " (n)" folder, never overwriting), parts
  "<name> - part N" (.mp4/.mov/.mkv kept, else .mp4), sound kept. Wheel: SPLIT, disabled
  for several files. Old SPLIT ½ removed (its dead message «هذا الفيديو أقصر من أن يُقسَم»
  went with it). Tests: `TestSplitVideo`, `TestSplitPoints`, `TestSplitHitTest`,
  `TestSplitCountText`, `TestWindowToolsOneVideo`.
- `a17f3ad` "1.0.1: Change Video Speed window replaces speed 2x": `ui/speeddlg.go` (silent
  playing preview, "Speed" label, 8 speed buttons 0.25× … 4× (slowest on the reading-start
  side, mirrored in Arabic; 2× chosen at start), Change Speed), host `speedPlay`,
  `speedStart`, `speedTitle`; `conv.SpeedVideo`, `conv.PlayPreview`, `atempoChain`. The
  preview is ONE FFmpeg run per play (`-ss`, setpts, fps=24, scaled+padded to the preview,
  raw RGBA on a pipe) paced by the reader; a new speed restarts it from the picture shown;
  at the end it starts again from 0. Output "<name>-<speed>x.mp4" (e.g. -0.5x, -2x),
  never overwriting. Sound keeps its pitch: atempo chained within 0.5..2 (0.25 = 0.5×0.5,
  3 = 2×1.5, 4 = 2×2). No sound in the video: a silent output, no error. Frame rate stays
  the video's own (`fps=` after setpts): faster drops frames, slower repeats them (0.25×
  looks like a slide show at the source rate, no interpolation). Progress title
  `launchVideoJobTitled` «جار تغيير السرعة إلى 0.5×». `ui.Visual` keeps "×" with its number
  like "%" (it was drawn "×0.5"). Wheel: SPEED, disabled for several files. Old SPEED 2×
  and its title «جار التسريع 2×» removed. Tests: `TestSpeedVideo`, `TestAtempoChain`,
  `TestPlayPreview`, `TestSpeedChips`, `TestSpeedTitle`, `TestVisualSpeedTitle`.
- Shots (local, not committed; `shots_windows_test.go` `TestShotsPhase3`,
  `TestShotsPhase3Speed`): `C:\Users\Glitch\Desktop\Falak phase 3 shots`.
- Test build: `build/phase3/Falak.exe` from `a17f3ad` (`vcs.modified=false`), 38,929,920
  bytes, SHA-256 `6a3c01c2da8854024f8c11846df1d174406bf4f216d55b2f10faf2e1d71e5b7e`.
  Other build folders and `build/SHA256SUMS.txt` untouched (verified).

### Phase 3 test results and decisions (owner, 2026-11-15)
- EVERYTHING PASSED on `build/phase3`, in Arabic and in English: the Split window (counter
  forms, points added / dragged / removed with × and Delete, exact parts in
  "<name> - parts" and " (1)"), the Speed window (silent preview, speeds, -0.5x with natural
  pitch, " (1)", extreme speeds), the Arabic progress line with the file name on the
  right, SPLIT and SPEED disabled for two videos.
- Decisions: the Speed window KEEPS 2× as the default speed. The removal of the two dead
  Arabic texts «هذا الفيديو أقصر من أن يُقسَم» and «جار التسريع 2×» is accepted.

### New features, phase 4 (2026-11-15; version stays 1.0.1) - the last video windows
- Approved Arabic (owner, 2026-11-15): "Smooth slow motion" «حركة بطيئة ناعمة», "Takes
  longer to process" «تستغرق المعالجة وقتًا أطول», "Crop Video" «اقتصاص الفيديو», "Crop"
  (button) «اقتصاص», "Aspect ratio" «نسبة الأبعاد», "Free" «حرّة», "W" «العرض», "H"
  «الارتفاع», "Cropping" (progress title) «جار الاقتصاص». The ratio labels stay numbers.
- `9831159` "1.0.1: smooth slow motion option in the speed window": a switch in the tray
  menu's style (40×24, accent when on, knob on the left when on in Arabic, like the menu)
  with its hint under it, on the side reading ends; `SpeedDialog.Smooth` (the user's
  choice, off at start), `SmoothEnabled` (only at 0.25×, 0.5×, 0.75×: `conv.SmoothApplies`
  via `Labels.SmoothOK`), `SmoothOn` (what the job uses). At 1.25× and faster the row is
  drawn dimmed with the switch off and can't be clicked; the choice is kept for when a
  slow speed is chosen again. Job: `conv.SpeedVideo(..., smooth, ...)`, filter
  `setpts=PTS/<speed>,minterpolate=fps=<source rate>` (FFmpeg 9.0.2's own minterpolate,
  defaults: motion-compensated, scene-change detection) instead of `fps=`; same length,
  frame rate, sound, name. The preview is NOT interpolated: at the preview's own size
  (896×504, 0.5×) minterpolate made ~17 pictures/s (28 s for 20 s of preview) on this
  32-thread machine, below the 24 the preview needs (minterpolate has no threading).
  MEASURED (1-minute 1080p30 H.264 from the owner's 4K sample, 0.5×, this PC): without
  the option 6 s, with it 588 s (~100×; single-threaded, so slower CPUs take longer).
  Tests: `TestSmoothSlowMotion` (plain output has a repeated frame, smooth has none),
  `TestSpeedSmooth`.
- `40e00b7` "1.0.1: Crop Video window": `ui/cropdlg.go` (`CropDialog`), `conv/cropvideo.go`
  (`VideoSize`, `CropVideo`, `MinCrop`), host in `videowin_windows.go` (`vkCrop`,
  `cropStart`; WM_CHAR, the caret timer and WM_ACTIVATE for the fields). Wheel item
  `{"v:crop", "CROP", "crop", one}` LAST in the video tools (8 tools; the existing crop
  icon), disabled for several videos, `OpensWindow`. The preview shows ONE frame from the
  middle of the video (the first is often black). The rectangle is in pixels of the
  picture AS SHOWN: `VideoSize` reads "WxH" and the display matrix rotation from FFmpeg's
  banner and swaps the sides for ±90° (FFmpeg autorotates before `-vf crop`, so the crop
  matches what the preview shows; tested with a rotated file). All numbers even
  (yuv420p). 8 handles (corners and side middles), outside dimmed, thirds lines, drag
  inside = move (clamped to the picture). Ratio buttons Free/1:1/16:9/9:16/4:3/3:4: a
  fixed ratio makes the largest rectangle of that ratio around the current centre; every
  handle keeps it (corner: the larger of the two pointer sizes; side: the other side
  follows around its centre). Smallest side 32 px (`MinCrop`; less for tinier videos):
  enforced silently, so NO "too small" message exists. W / H fields: click = focus with
  the number selected (first digit replaces it), digits / Arabic-Indic digits /
  backspace, at most 5 digits, caret blinks; applied (even, clamped, around the centre,
  the other side following a fixed ratio) when another control is pressed, the window
  loses focus or on Enter (Enter then crops). Esc / X close, Enter crops. Job: `crop=w:h:x:y`,
  x264 CRF 16 like Trim, `-c:a copy`; if the copy fails (a sound format the output can't
  hold, e.g. WMA in MP4) it runs once more with AAC 192k. Output "<name>-cropped" (.mp4,
  .mov, .mkv kept, else .mp4), " (1)" series. Progress title «جار الاقتصاص». Tests:
  `TestCropVideo` (exact area by pixels, sound copied, " (1)", odd sizes made even),
  `TestVideoSizeRotated`, `TestCropDrag`, `TestCropFields`, `TestCropHitTest`,
  `TestCropNumbersDontFlip` (Arabic ratio labels, W/H numbers and speed values each look
  most like the same label in English: "16:9" never "9:16"; a deliberately swapped label
  was caught), `TestWindowToolsOneVideo` (now with v:crop).
- Gate note: `TestProgressMoves` (conv, timing: fails if a progress update pauses over 2 s)
  failed TWICE this round, each time in a slow full run (conv ~400 s instead of ~105 s;
  machine busy): image to PDF paused 3.4 s before the crop commit, 7Z 2.1 s before the
  handover commit. Never in a normal-speed run: `67370cf` passed 2 of 2, `40e00b7` 2 of 2
  (separate worktree), and the gates passed before each commit. If it fails, rerun and
  compare the conv time before suspecting the code.
- Shots (local, not committed; `shots_windows_test.go` `TestShotsPhase4`):
  `C:\Users\Glitch\Desktop\Falak phase 4 shots` (Speed with the switch on / off /
  disabled, Crop Free / 16:9 / 9:16 / typing in W, the Cropping progress window, the
  tools wheel with CROP; Arabic and English, light and dark).
- Test build: `build/phase4/Falak.exe` from `40e00b7` (`vcs.modified=false`), 38,981,120
  bytes, SHA-256 `4d1498feabaca889e2acdc94c2d118245d58434af89f4a2177bb236fe61f46dc`.
  Other build folders and `build/SHA256SUMS.txt` untouched (verified, `sha256sum -c` OK).

### 1.0.1 build (2026-11-09, from `961a735`; not published)
Falak 1.0.1 = 1.0.0 plus the full-screen fix:
- `ad92427` "1.0.0: log full-screen block details": when the wheel is blocked, the log
  (once per drag) shows the `SHQueryUserNotificationState` value, the foreground window
  (class, process, rect, styles incl. topmost / layered / transparent), the monitor rect and,
  on QUNS_BUSY, the first visible window covering the monitor (`EnumWindows`; minimized,
  cloaked and Falak's own windows skipped). Code: `fullscreen_windows.go`.
- `bb39861` "1.0.0: QUNS_BUSY alone no longer blocks the wheel": a user's PC (NVIDIA Overlay,
  Avast, Avira) reported QUNS_BUSY while he was only in File Explorer, so the wheel never
  opened. States 3 (D3D full screen) and 4 (presentation) still block; any other state
  blocks only when the foreground window is a real full-screen window: no WS_THICKFRAME,
  covers the monitor, not explorer.exe, not Progman / WorkerW, not click-through
  (WS_EX_TRANSPARENT, or WS_EX_LAYERED + WS_EX_TOPMOST), not one of Falak's own windows
  (this process or the helper, same exe). A QUNS_BUSY that is ignored is logged with the same
  details ("QUNS_BUSY ignored"). Rule: `blockDecision` in `fullscreen_rule_windows.go`,
  19 cases in `fullscreen_rule_windows_test.go`.
- `961a735` "1.0.1: bump version": `appVersionNum` "1.0.1" (menu subtitle), VERSIONINFO 1.0.1.0
  (`rsrc_windows_amd64.syso` patched in place: only 4 bytes changed, the two FILEVERSION /
  PRODUCTVERSION low DWORDs and the two "1.0.0.0" strings; `res/raha.rc` matches), installer,
  portable script, audit, verify-install.ps1, README / README.txt (two owner-approved Arabic
  lines, version number only), THIRD_PARTY_LICENSES.txt, HANDOVER.md, CLAUDE.md rules.

Built from a clean tree at `961a735` (`vcs.revision=961a735eafc844e371e7776fcc3a4177a3d4eb62`,
`vcs.modified=false`; `shots_windows_test.go` moved out during the build), after all gates
passed. Windows reads FileVersion / ProductVersion 1.0.1.0 from Falak.exe and the Setup; the
icon and manifest resources are byte-identical to 1.0.0 (only the version resource differs).
`build/SHA256SUMS.txt` holds the three hashes. The 1.0.0 files were removed from `build/`
(they are on the GitHub v1.0.0 release).
| File | Size (bytes) | SHA-256 |
|---|---|---|
| `build/Falak.exe` | 38,710,784 | `963ab4e449127ac86ccbd4042c4f075c9acc127d3a7d27572befdb4ee84f820f` |
| `build/FalakSetup-1.0.1.exe` | 12,594,783 | `89da85bd78a89fd1b97b1f498853b8dc795be56309f2b660a08067a5110467d9` |
| `build/Falak-1.0.1-portable.zip` | 16,932,024 | `b89a64202156559bc1b9f9246f3496255a4c65665a2267d90891b50881c14398` |

Checked: the zip holds exactly `Falak.exe`, `README.txt`, `THIRD_PARTY_LICENSES.txt`; the
`Falak.exe` in the zip and in the Setup payload (7-Zip) are byte-identical to
`build/Falak.exe`; `go list -deps .` has no shots code. Do not rebuild for docs-only commits.

### FINAL build (2026-11-09, from `57f35db`; not published)
Built from a clean tree at `57f35db` (`vcs.revision=57f35db8821db9a9a8015dc47fff6b14d7ffff7a`,
`vcs.modified=false`; the untracked `shots_windows_test.go` was moved out of the repo during
the build), after all gates passed. Version 1.0.0 (VERSIONINFO 1.0.0.0 in Falak.exe and the
Setup). `build/SHA256SUMS.txt` holds the three hashes.
| File | Size (bytes) | SHA-256 |
|---|---|---|
| `build/Falak.exe` | 38,699,008 | `41374c553b7e3a0e11ba5e179f1b4a424a33fa5385667096f867ef970a1c1d7c` |
| `build/FalakSetup-1.0.0.exe` | 12,606,153 | `8a44dc4c52252f48a2559701b978cdb03fe30b2702f70fc931937c58c6e6da2c` |
| `build/Falak-1.0.0-portable.zip` | 16,926,635 | `a42d03a2cc3bb0f7c39f0887954cf05a4b716392e3d9eb8883c645e95d2d5926` |

Checked: the zip holds exactly `Falak.exe`, `README.txt`, `THIRD_PARTY_LICENSES.txt`; the
`Falak.exe` in the zip and in the Setup payload (7-Zip) are byte-identical to
`build/Falak.exe`; `go list -deps .` has no shots code. **Do not rebuild**: this is the final
build; later commits (like the one recording it) change documentation only.

### Commits since the import
| Commit | Title | Purpose |
|---|---|---|
| `e29e934` | import current state into git | First commit of the handed-over 1.0.0 source (old history is not in this repo). |
| `30476e0` | uninstaller reads the keep-settings answer correctly | `MessageBox … IDYES` jump instead of `Pop $0` (it read a leftover nsExec code); every nsExec result popped. |
| `fa55486` | uninstaller removes only the files it installed | No `RMDir /r "$INSTDIR"`; deletes its own files, `RMDir` only when empty; audit check. |
| `8d56baf` | video compress no longer blocks other conversions | `engineMu` held only for the FFmpeg download, not the whole encode. |
| `7a081a1` | handover HEIC notes match the code | HEIC input = gen2brain/heic (pure-Rust crate, WASM→Go); output = FFmpeg libx265 + own HEIF container. |
| `fc5e411` | video compress-to-size works with current FFmpeg | `-pass:v` / `-passlogfile:v` (plain `-pass 2` hit the AAC stream). |
| `657d478` | compress leaves no two-pass logs in .falak-tmp | Removed `-0.log*` logs; test checks `.falak-tmp` is gone. |
| `ae9fbcb` | GIF progress test uses a 9 s clip | Test was timing-flaky on fast machines. |
| `7aed3ae` | key helper asks UAC each time; no scheduled task | Removed silent-elevation task `Falak\KeysHelper`; helper/installer/uninstaller delete an old one. |
| `256b8da` | NSIS script compiles with makensis on Windows | `${__FILEDIR__}\…` backslashes (Windows makensis 3.13 failed on `/`); audit check. |
| `b8bbda8` | uninstaller without PowerShell; Falak closed by message | NSIS `EnumRegValue` loop for Layers; `wmQuitUpdate` (0x8009) to `FalakMsg`, wait 5 s, then `taskkill /F`, +3 s for the helper. |
| `8bd429d` | helper prompt says approval is asked each time | Owner text: EN "Windows will ask for your approval each time the helper starts." AR «سيطلب ويندوز موافقتك في كل مرة يعمل فيها المساعد.» |
| `3c96dbb` | engines pinned to fixed releases with SHA-256 checks | `conv/engines.go`, `falak-engine.txt`, hash check, `enginesnet` test. |
| `fae66c4` | engines live in %LOCALAPPDATA%\Falak\tools | Engines out of the install folder; old `Programs\Falak\tools` removed. |
| `c5be6f9` | one install folder when the user picks their own | `installDir()` = installer's InstallLocation; old second copy removed; `InstallDirRegKey`. |
| `af994d0` | vendor the dependencies; builds from a fresh clone | `vendor/` + `go.sum`, no `../deps`; all deps proven unmodified upstream; 3 version replaces; `godebug default=go1.24`; test images in `conv/testdata`. |
| `8096641` | docs match the review changes | README (English part), notices, licenses (x/sync dropped), handovers. |
| `58c30f9` | CLAUDE.md with the permanent rules | Rule file at repo root. |
| `04bb412` | Arabic README matches the review changes | Owner's Arabic: uninstall sentence, helper section, pinned-engines sentence. |
| `b759137` | portable README.txt in the repo | `installer/portable/README.txt`, owner's EN+AR text. |
| `50e2895` | build step for the portable zip | `installer/make_portable.py` (deterministic zip of exactly 3 files) + audit check. |
| `b57ba4f` | CLAUDE.md release-folder rule: read only with permission | Both release-folder paths listed. |
| `1df38b2` | compress to a set size works in watched folders; failures shown | Pass logs in `%TEMP%\falak-pass-*`; `completePassLogs` renames x264's `*.temp` logs with retry; FFmpeg errors logged (args + last 15 stderr lines) and the specific line shown; failed jobs also show a message box. |
| `8439bc2` | the target size box can be focused and edited | Focus, caret, selection, Enter; default 100 MB or half the original (≤ 100 MB). |
| `4f45c70` | no leftovers in .falak-tmp after failed or interrupted jobs | Delete/move retry 5 s; `temp-folders.txt` record; start-up cleanup of stale `<UnixNano>-<n>…` entries only. |
| `d122770` | helper menu item shows whether the helper really runs | Toggle = helper running; refusing UAC turns the setting off. |
| `405e2a7` | once-per-session hint when a desktop drag needs the helper | Owner text (see below), tray balloon. |
| `b98e64c` | log every helper hint/offer attempt and tray notification | Logs Shell_NotifyIcon result + notification settings (read only). |
| `26f752d` | helper hint also shown in a Falak window | Message box in addition to the balloon (balloons are silenced on the owner's machine). |
| `53fee02` | first run also opens the How to use card | Only on the very first run, never with `--startup`. |
| `37d6258` | background job done: the progress window shows Done briefly | 2.5 s, no focus, X / new drag close it, not over the wheel or another panel. |
| `312d695` | session handover | This section (2026-11-08/09). |
| `aecd928` | compress to a set size works before FFmpeg is downloaded | The dialog read the length with FFmpeg at open; on a fresh install it was 0 and every target was "too small". `conv.TargetTooSmall`: unknown length is never too small; the job checks again after the download. |
| `c504236` | compress to a set size counts MB like File Explorer | `ui.BytesPerMB = 1,048,576` for "smaller than the original", the red hint and the default offer (the bitrate already used it). Answer to Q4. |
| `3bdb5b0` | a finished file never replaces a file that already exists | `finishTemp` moves with `renameNoReplace` (Windows `MoveFileEx` without replace) and continues the start-time " (n)" series (`nameSeries`); output helpers return the real path. Answer to Q5a. |
| `7d67d54` | handover records portable downgrade and wmQuitUpdate as deferred | Q5b, Q5c. |

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
- 2026-11-09 (builds from `37d6258` and `aecd928`):
  - First-run "How to use" card: opens only on the first run; not after Quit and restart,
    not with Start with Windows.
  - Uninstall answering «لا» deletes the settings (`HKCU\Software\Falak`, including
    `Welcomed`) and the tools (`%LOCALAPPDATA%\Falak\tools`).
  - The "Done" window comes back after a background job, closes by itself, does not steal
    the focus and plays no new sound.
  - Compress to 40 MB on a fresh install without FFmpeg worked (`aecd928`): FFmpeg was
    downloaded first, output 38 MB.
  - Custom folder `C:\FalakTest`: the uninstaller removed only Falak's files and kept
    `keep.txt`; answering «نعم» kept the settings (`Welcomed`), and the card did not open
    on reinstall.
- 2026-11-09 (Setup built from `da1d3f1`, installed over the previous install):
  - **MB unit (`c504236`) PASSED.** Input `G:\NVIDIA\Desktop\Desktop 2026.11.06 -
    19.45.12.04.DVR.mp4` (Explorer: 135,027 KB). The compress dialog showed "Original size:
    131.9 MB" (matches Explorer, not 138). Target 132: red "Enter a size smaller than the
    original file", Compress disabled. Target 131: no message, Compress enabled (not pressed).
  - **No overwrite (`3bdb5b0`) PASSED.** During an MP4 -> WMV conversion of the same file, a
    1 KB placeholder `Desktop 2026.11.06 - 19.45.12.04.DVR.wmv` was created in the same
    folder. Afterwards the placeholder was untouched (1 KB) and the output was saved as
    `Desktop 2026.11.06 - 19.45.12.04.DVR (1).wmv` (157,024 KB). Both test .wmv files were
    deleted afterwards; the original .mp4 is kept.
- 2026-11-09, FINAL build (`57f35db`): Setup hash verified (`8a44dc4c…6e6da2c`); MP4 -> MP3
  works; compress to 40 MB gave 38.7 MB (40,592,005 bytes); the How to use card shows the OK
  button and no duplicate title.
- VirusTotal, `FalakSetup-1.0.0.exe`: **1/71**, Microsoft: Undetected. Only Trapmine flagged
  it (`Malicious.moderate.ml.score`, an ML false positive). **No new Microsoft report is
  needed.** Report:
  https://www.virustotal.com/gui/file/8a44dc4c52252f48a2559701b978cdb03fe30b2702f70fc931937c58c6e6da2c

### Owner's machine notes
- Notifications are disabled by Chris Titus Tech WinUtil:
  `HKCU\Software\Policies\Microsoft\Windows\Explorer\DisableNotificationCenter = 1` and
  `HKCU\Software\Microsoft\Windows\CurrentVersion\PushNotifications\ToastEnabled = 0`.
  Tray balloons never show here: never rely on balloons alone for anything the user must see.
  **Never change the owner's Windows settings** (read only, as `notificationState()` does).
- `HKCU\Software\Falak` has `Welcomed` (kept by the «نعم» uninstall test), so the next start
  is not a first run.
- After a restart the wheel appears on desktop drags **without** the helper (log: drag from
  explorer.exe, `held=true dragging=true rawL=0x8000`, wheel shown). So the helper hint
  window (`26f752d`) cannot be tested on this machine.
- SteelSeries Sonar is installed and may be why the wheel appear sound is sometimes not
  heard (the stream closes after 20 s idle; `appear.wav` is 100 ms). **No sound change was
  made.**
- Defender real-time protection is on; Controlled Folder Access is off.
- Drive G: (NTFS, "My Archive") holds NVIDIA recordings in `G:\NVIDIA\Desktop`; programs
  there briefly lock new files.

### DONE
- Code review findings fixed (uninstaller, compress lock, helper elevation, PowerShell,
  pinned engines, one install folder, vendoring, temp leftovers, compress in watched
  folders, editable size box, helper state/hint, first-run card, background "Done").
- 2026-11-09: compress to a set size before FFmpeg is downloaded (`aecd928`); one MB unit
  (Q4, `c504236`); a finished file never replaces an existing one (Q5a, `3bdb5b0`); Q5b/Q5c
  recorded as deferred (`7d67d54`); old `deps` folder deleted (Q6); open questions answered.
- Manual tests of the MB unit (`c504236`) and no-overwrite (`3bdb5b0`) passed on the owner's
  machine (details above).
- "How to use" card: the OK button was never drawn (gg keeps the clip after `Pop()`, see
  Mistakes); fixed with `dc.ResetClip()` in `ui/helpcard.go` (commit "help card draws its OK
  button"), test `ui/helpcard_test.go`. The click area (`HitTest` = 2) was always there and
  matches the drawn button. Other `Clip()` uses checked: `ui/panel.go` already calls
  `ResetClip()` after `Pop()`, `ui/wheel.go` resets the clip without `Push`/`Pop`; the
  `Push`/`Pop` in `ui/icons.go` and `ui/menu.go` have no clip.
- Progress window mirrors in Arabic (`88a8c4a`); Arabic shaping keeps non-joining letters
  (a hamza U+0621 after a joining letter became a NUL, `55e1923`); help card drops the duplicate title line
  (`2668505`); tanween kasra (U+064D) removed from all Arabic text (`a2c793c`, owner-approved).
- README screenshots (commit "README screenshots"): 20 PNGs in `docs/` (see Section 7), shown
  in both README sections with English alt text; old `docs/tray-menu.png` and
  `docs/compress-dialog.png` removed.
- Final build from `57f35db` (2026-11-09), hashes above; manual test on the owner's PC and
  VirusTotal done (results above); Microsoft report not needed. RELEASE_NOTES.md has the
  checksums, the VirusTotal link and the SmartScreen line; README.md links the scan.
- GitHub release v1.0.0 published (2026-11-09); old GitHub main kept as `old-upload`.
- Docs: README (EN + owner's AR), notices, licenses, handovers, CLAUDE.md, portable README.txt.
- Owner's manual tests listed above.
- 1.0.1: full-screen fix (`ad92427`, `bb39861`), version bump (`961a735`), build from
  `961a735` (hashes above).

### IN PROGRESS
- New features, phases 1-4 done (see above); the owner tested `build/phase3` (everything
  passed); waiting for the test of `build/phase4/Falak.exe` (Smooth slow motion, Crop
  Video window). All five video windows are built. `build/` still holds the 1.0.1
  build from `961a735` (not published; it predates the new features, so a release with them
  needs a new build). The user who had the blocked wheel confirmed 1.0.1 fixes it
  (2026-11-15). GitHub release postponed until new features are added (owner's decision).

### NEXT
1. Owner's manual test of 1.0.1.
2. DONE (2026-11-15): 1.0.1 was sent to the user who had the blocked wheel; he confirmed it
   fixed the problem (the wheel now opens). His new `falak.log` was not reviewed in a session.
3. VirusTotal scan of the 1.0.1 files.
4. GitHub release v1.0.1: POSTPONED (owner's decision 2026-11-15). New features come first;
   publishing happens later, only on the owner's decision. Before that release: update the
   README VirusTotal link to the 1.0.1 scan and write release notes for 1.0.1.
5. Owner's manual test of `build/phase4/Falak.exe` (phase 4: Smooth slow motion, Crop).
6. README, all at once before publishing (owner's decision): archive -> PDF, the compress
   folder, TRIM, the Snapshots window; its Arabic part needs the owner's text.
7. The video windows are complete (Trim, Snapshots, Split, Speed, Crop). README needs them
   (item 6).

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

### Answers to the open questions (owner, 2026-11-09)
1. Publishing: when we publish, first keep the current GitHub `main` in a backup branch
   `old-upload`, then make this local history `main`. **Done on 2026-11-09** (see GitHub above).
2. Git identity: `UncleGlitch <135931273+GlitchGLH@users.noreply.github.com>`.
3. The 47 MB vendored repository is accepted.
4. Done: compress to a set size and its checks use 1 MB = 1,048,576 bytes, like File
   Explorer (`c504236`).
5. (a) Done: a finished file never replaces an existing one (`3bdb5b0`). (b) portable
   downgrade and (c) `wmQuitUpdate`: deferred, see "Known and deferred".
6. Done: `C:\Projects\Falak\deps` deleted (nothing in the build or tests used it; backup:
   `C:\Users\Glitch\Desktop\Falak-handover\falak-deps.zip`). `vendor/` untouched.

### Environment
- Go **go1.27.3 windows/amd64** (`C:\Program Files\Go`), installed with winget
  (`GoLang.Go`). Module says `go 1.25.0` with `godebug default=go1.24`.
- NSIS **3.13** (`C:\Program Files (x86)\NSIS`), installed with winget (`NSIS.NSIS`).
- Python **3.14.10**, run as `py` (`python` is the Store alias and does not work in Git Bash).
- go-winres not installed (the `.syso` is checked in).
- `C:\Projects\deps` junction: created for the first test run, **removed after vendoring**
  (`af994d0`); builds and tests no longer need it. The old copies in `C:\Projects\Falak\deps`
  were deleted on 2026-11-09 (backup `C:\Users\Glitch\Desktop\Falak-handover\falak-deps.zip`).
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
- README screenshots: `shots_windows_test.go` (repo root, build tags `windows && shots`, so it
  is never in Falak.exe nor in the normal tests) renders 20 PNGs off-screen with the app's
  own drawing code, fonts and texts (it fails if a copied label pair is not in the app
  source). Output `$SHOTS_OUT` (default `C:\Projects\falak-shots`), 200 % scale, sample
  `Holiday.mp4` / 48.3 MB:
  `go test -tags "shots wasm2go nodynamic" -run TestShots -count=1 .`
  The file is local and not committed (owner's decision); the PNGs it made are in `docs/`.

### Mistakes and dead ends (do not repeat)
- Never `taskkill /IM ffmpeg.exe` (all FFmpeg processes) to clear a test run: it would also
  kill a conversion the owner's own Falak is running. Stop only the test's own process.
- `bc` is not in Git Bash here; time with `date +%s` arithmetic.
- Judging Arabic/Latin order by LOOKING at a rendered PNG is unreliable for the model (it
  "reads" the line in logical order). Check order with pixels: crop each end, or match a
  render of the Latin part alone against the line (see `TestCardFileNameFirstArabic`).
- Testing a drop feature only through `conv.Run` misses the drop path: the wheel offers
  the FIRST file's item IDs and `applicable`/`planDrop` filter the selection. Test mixed
  selections through `planDrop` + `runDropFiles` (see `TestDropCompressMixed`).
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
- **gg keeps the clip after `Pop()`**: `github.com/fogleman/gg`'s `Pop()` restores the state
  but keeps the current mask, so after `Push(); …; Clip(); …; Pop()` everything drawn later
  is still clipped. Call `ResetClip()` after `Pop()` (as `ui/panel.go` and now
  `ui/helpcard.go` do). This hid the help card's OK button. Do not edit `vendor/`.
- Checking a diff for Arabic lines with Python on Windows: read
  `sys.stdin.buffer.read().decode("utf-8")`; plain `sys.stdin.read()` uses the ANSI code
  page, matches no Arabic and silently reports 0.

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
  («صيغة GIF تدعم الفيديو حتى 10 ثوان فقط») and reports two-pass progress (0 → 0.4 palette pass → 1.0).
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
- GIF refusal: «صيغة GIF تدعم الفيديو حتى 10 ثوان فقط»
- T('Falak is running', 'فلك يعمل الآن') · T('Drag a file and press Shift while dragging.', 'اسحب ملفًا واضغط Shift أثناء السحب.')
- T('Conversion failed', 'تعذّر التحويل') · T('Done', 'اكتمل التحويل') · T('Compressing', 'جار الضغط')
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
no captures) · `help_windows.go` (How-to-use host) · `msgcard_windows.go` (message cards
replacing MessageBox) · `win32_windows.go` (Win32 decls) ·
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
`StartActive`) · `helpcard.go` · `msgcard.go` (message card) · `splitdlg.go` (Split Video) · `speeddlg.go` (Change
Video Speed, Smooth slow motion switch) · `cropdlg.go` (Crop Video) · `popanim.go` (the wheel's open/close for
popups) · `icons.go` (supersampled line icons) · `fonts.go` (Inter/Amiri)
· `blur.go` (FrostBackground, GaussianBlur, box blurs) · `textblock.go` ·
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

**`docs/`**: the README screenshots (Section 7). The older `wheel-light-dark.png` was removed
on 2026-11-09 (commit "remove unused wheel-light-dark.png").

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
local `deps/` copies were retired — commit `af994d0`). The former "v0.0.0" pins
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
commit `e29e934` holds the same sound files and call sites): `sound_windows.go`, `res/*.wav`,
and every call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` / `soundOn`.
No redesign, no "improvements", no volume changes — if a sound task ever comes up, **restore
from `e29e934`** (`git checkout e29e934 -- sound_windows.go res/`), do not edit.

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
- **Key helper = UAC prompt every start** (`7aed3ae`, text `8bd429d`): the scheduled task
  `Falak\KeysHelper` ran a user-writable exe with highest privileges and no prompt (silent
  elevation). No task any more; the elevated helper, the installer and the uninstaller delete
  an old one (installers fall back to one `runas` prompt). After a refused prompt the helper
  is not relaunched on every drag.
- **Engines pinned** (`3c96dbb`): FFmpeg 9.0.2 `full_build-shared` from GyanD/codexffmpeg
  (permanent numbered releases; BtbN "latest" broke compress-to-size: `-pass 2` hit the AAC
  stream, fixed with `-pass:v`/`-passlogfile:v` in `fc5e411`), PDFium chromium/8138,
  qpdf 12.3.2, resvg 0.49.0, each with SHA-256; a mismatch reuses the existing "download
  didn't finish" text. Never go back to "latest" links.
- **Engines folder** `%LOCALAPPDATA%\Falak\tools` (`fae66c4`), not the install folder.
- **One install folder** (`c5be6f9`): `installDir()` = installer's InstallLocation when
  Falak.exe is there, else `%LOCALAPPDATA%\Programs\Falak`; a second copy left by old builds
  in the default folder is removed; `InstallDirRegKey` reuses the folder on update.
- **Uninstaller**: deletes only the files it installed and `RMDir` (no `/r`) on `$INSTDIR`
  (`fa55486`); reads the keep-settings answer via `MessageBox … IDYES` (`30476e0`); no
  PowerShell — Layers cleanup is an NSIS `EnumRegValue` loop; closes Falak with
  `wmQuitUpdate` (0x8009) to `FalakMsg`, waits 5 s, then `taskkill /F`, then up to 3 s more
  for the elevated helper (`b8bbda8`).
- **Video compress** no longer holds `engineMu` during the encode (`8d56baf`).
- **Vendored dependencies** (`af994d0`), see Section 3.

## 7. Screenshots and docs images
Since 2026-11-09 the README shows 20 PNGs in `docs/`, rendered off-screen on Windows with the
app's own drawing code and fonts (renderer and command: Section "Current state", Environment):
`wheel-light/dark`, `tools-wheel-light/dark` (shared by both languages: the wheel has no
translated text), and `tray-menu-`, `compress-dialog-`, `progress-`, `how-to-use-` each as
`-en-light`, `-en-dark`, `-ar-light`, `-ar-dark`. 200 % scale, sample `Holiday.mp4` / 48.3 MB,
about 3.2 MB in total. The older `docs/tray-menu.png` and `docs/compress-dialog.png` were
removed (on GitHub they are the owner's real screenshots; publishing replaces them).

Older notes (`docs/wheel-light-dark.png` was removed on 2026-11-09; kept for history): its reproduction: `cmd/themeprev` renders four wheel sets
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
- Repository: `C:\Projects\Falak`, git `main`; history starts at `e29e934` (import of the state
  after `7bdc834` "1.0.0: no seam in the hover fill mid-animation"); later commits are the
  review fixes listed in Section 6. Version still 1.0.0.
- New builds of Falak.exe / FalakSetup-1.0.0.exe exist only in `build/` (not published; their
  hashes differ from the ones under review). Manual testing on real Windows is pending.
- **GitHub repo github.com/GlitchGLH/Falak is public (MIT, author UncleGlitch).**
  (2026-11-09: the README now uses the new screenshots in `docs/`, see Section 7.)
- (2026-11-09: release v1.0.0 is published, see the Current state section.)
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
> modify the sound code (sound_windows.go, res/*.wav, sound call sites — frozen; same as e29e934);
> (3) never change an existing Arabic string; (4) one commit per task with only
> "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"; (5) version stays 1.0.0;
> (6) run the gates (go test ./..., go vet with tags, installer/audit_paths.py) before every
> commit. The release v1.0.0 on github.com/GlitchGLH/Falak is published (2026-11-09).
