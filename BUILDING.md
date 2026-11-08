# Building Falak

## Requirements

- Go 1.25 or newer (the module says `go 1.25.0`; GODEBUG defaults are kept
  at Go 1.24 with `godebug default=go1.24`)
- Windows for the app itself (the `ui` and `conv` packages also build on other
  systems, which is what the tests use)
- NSIS 3.x (`makensis`) for the installer

## Dependencies

Every dependency is vendored in `vendor/` (`go mod vendor`), so a fresh clone
builds offline, with no extra folders. All of them are unmodified upstream
releases or commits; `go.sum` records their hashes. Three are pinned with
`replace` to the exact versions Falak is built from, because other
dependencies ask for newer ones:

- `golang.org/x/sys` v0.30.0
- `github.com/ebitengine/purego` v0.8.4
- `github.com/andybalholm/brotli` v1.1.0

To change a dependency: edit `go.mod`, then run `go mod tidy` and
`go mod vendor`, and commit `go.mod`, `go.sum` and `vendor/` together.

## Build

    GOOS=windows GOARCH=amd64 go build -tags "wasm2go nodynamic" -ldflags "-s -w -H windowsgui" -trimpath -o build/Falak.exe .

## Tests

    go test ./...

FFmpeg-based tests use an `ffmpeg` on PATH and skip without one.

## Version resources

`rsrc_windows_amd64.syso` is prebuilt and checked in; the linker picks it up
automatically. `res/raha.rc` mirrors its contents for reference.

## Installer

1. Install NSIS 3.x (`makensis`).
2. Copy a built `Falak.exe` into `installer/`.
3. Run:

       makensis -INPUTCHARSET UTF8 installer/falak.nsi

   The script resolves its assets (icon, license, Falak.exe) relative to its
   own directory and writes the setup exe two levels up.
