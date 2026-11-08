# Building Falak

## Requirements

- Go 1.24 or newer
- Windows for the app itself (the `ui` and `conv` packages also build on other
  systems, which is what the tests use)

## Dependencies

All Go dependencies are pinned with `replace` directives in `go.mod` to a
local `deps/` directory that must sit NEXT TO this folder (its parent):

    falak-build/
    ├── falak-src/   <- this repository
    └── deps/        <- the pinned dependency copies

This is required, not a style choice: several dependencies are pinned to
versions that do not exist upstream (`gen2brain/avif v0.0.0`,
`gen2brain/heic v0.0.0`, `gen2brain/webp v0.0.0`, `srwiley/oksvg v0.0.0`,
`srwiley/rasterx v0.0.0`) because they are wasm2go builds generated for this
project. `go mod tidy` against the public module proxy cannot resolve them
(it fails with `unknown revision v0.0.0`), so the standard public modules
cannot replace them as-is.

## Build

    GOOS=windows GOARCH=amd64 go build -tags "wasm2go nodynamic" -ldflags "-s -w -H windowsgui" -trimpath -o Falak.exe .

## Tests

    go test ./...

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
