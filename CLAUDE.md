# Falak — rules for Claude

Read `CLAUDE_HANDOVER.md` before any work: it has the project map, build
commands, decision history and current state.

## Permanent rules

- **Never touch the sound code**: `sound_windows.go`, `res/*.wav` and every
  call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` /
  `soundOn`. No fixes, no "improvements". If a sound task comes up, restore
  from commit `8816700`, never edit.
- **Arabic text**: never change an existing Arabic string (code, installer,
  README). New or replacement Arabic only with the exact text the owner
  gives; show every changed Arabic line in full, before and after. If an
  Arabic sentence becomes inaccurate, show it, propose a replacement and wait.
- **Commits**: one commit per task, message `1.0.0: <what changed>`, ending
  with exactly `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`;
  no Claude-Session or "Generated with" line.
- **Files**: UTF-8 with LF line endings (`vendor/` stays as upstream ships it).
- **Version** stays 1.0.0 until the owner says otherwise.
- **Gates before every commit**: `go test ./...`,
  `GOOS=linux go vet -tags "wasm2go nodynamic" ./...`,
  `py installer/audit_paths.py`.
- **Builds** go to `build/` only. Build the Setup only when the owner asks.

## Temporary rule

- **Never touch `Falak-1.0.0-release`** (read, move, rebuild or overwrite):
  a Microsoft false-positive review on the hashes of its files is pending.
