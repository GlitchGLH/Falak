# Falak — rules for Claude

Read `CLAUDE_HANDOVER.md` before any work: it has the project map, build
commands, decision history and current state.

## Permanent rules

- **Language of messages**: every message to the owner is written in
  Arabic: progress notes while working, questions and final reports alike.
  The owner's prompts may be in English; the reply is still in Arabic.
  (Code, comments, commit messages and English docs stay in English; Arabic
  product text still needs the owner's approved wording, see below.)
- **Never touch the sound code**: `sound_windows.go`, `res/*.wav` and every
  call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` /
  `soundOn`. No fixes, no "improvements". **Never add, remove or move a sound
  call**, and no change may make a sound play where none plays today. If a
  sound task comes up, restore from commit `e29e934`, never edit.
- **Arabic text**: no new or changed Arabic text without the owner's exact
  approved text (code, installer, README, README.txt). Show every changed
  Arabic line in full, before and after. If an Arabic sentence becomes
  inaccurate, show it, propose a replacement and wait.
- **Commits**: one commit per task, message `2.0.0: <what changed>`, ending
  with exactly `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`;
  no Claude-Session or "Generated with" line.
- **Files**: UTF-8 without BOM, LF line endings (`vendor/` stays as upstream
  ships it).
- **Version** stays 2.0.0 until the owner says otherwise.
- **Gates before every commit**: `go test ./...`,
  `GOOS=linux go vet -tags "wasm2go nodynamic" ./...`,
  `py installer/audit_paths.py`.
- **Builds** go to `build/` only. Build the Setup only when the owner asks.
- **Never change the owner's Windows settings** (notifications, policies,
  Defender, registry outside Falak's own keys). Read them only.
- **Never rely on a tray balloon alone** for anything the user must see:
  notifications are off on the owner's machine (and for many users); show
  it in a Falak window too.
- **Do not implement the rejected proposals 3 and 4** (see
  `CLAUDE_HANDOVER.md`, "Rejected by the owner").

## Temporary rule

- **`Falak-1.0.0-release` must never be modified** (no writing, moving,
  extracting into it, rebuilding or overwriting), and may be **read only with
  the owner's explicit permission**: a Microsoft false-positive review on the
  hashes of its files is pending. Both copies:
  - `C:\Users\Glitch\Desktop\Falak-1.0.0-release`
  - `C:\Users\Glitch\Desktop\Falak with Claude Code\Falak-1.0.0-release`
