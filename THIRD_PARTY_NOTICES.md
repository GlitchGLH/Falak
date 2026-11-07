# Third-party notices

## Libraries compiled into Falak (Go modules, kept in deps/)

| Library | License |
|---|---|
| golang.org/x/sys, x/image, x/text, x/net, x/sync | BSD-3 (The Go Authors) |
| github.com/fogleman/gg | MIT |
| github.com/golang/freetype | FreeType License / MIT (dual, your choice) |
| github.com/bodgit/sevenzip, bodgit/plumbing, bodgit/windows | BSD-3 |
| gen2brain/avif, gen2brain/heic, gen2brain/webp | MIT (Go wrappers; see the WASM note below for the libraries embedded in each) |
| github.com/klauspost/compress | BSD-3 |
| github.com/andybalholm/brotli | MIT (Brotli Authors) |
| github.com/pierrec/lz4/v4 | BSD-3 |
| github.com/ulikunitz/xz | BSD-3 |
| github.com/nwaples/rardecode/v2 | BSD-3 |
| github.com/srwiley/oksvg, srwiley/rasterx | BSD-3 |
| github.com/ebitengine/purego | Apache 2.0 |
| github.com/tetratelabs/wazero | Apache 2.0 |
| go4.org | Apache 2.0 |
| github.com/hashicorp/errwrap, go-multierror, golang-lru/v2 | MPL-2.0 |
| test-only: stretchr/testify, go-difflib, go-spew, objx, yaml.v3 | MIT / ISC / Apache 2.0 |

Full license texts ship next to each module in the deps/ folder. A copy of every compiled module's license text ships with the app as `THIRD_PARTY_LICENSES.txt` (next to Falak.exe, and in the portable zip).

## Programs Falak downloads at first use (never bundled)

| Tool | License | Downloaded from |
|---|---|---|
| FFmpeg (BtbN win64 build) | **GPL** — see warning | https://github.com/BtbN/FFmpeg-Builds (the BtbN builds, NOT ffmpeg.org) |
| PDFium (bblanchon binaries) | BSD-3 (PDFium) | https://github.com/bblanchon/pdfium-binaries |
| qpdf 12.3.2 | Apache 2.0 | https://github.com/qpdf/qpdf |
| resvg | Apache 2.0 / MIT | https://github.com/linebender/resvg |

**License warning (please read):** the FFmpeg build Falak downloads is the
**GPL** build. GPL places conditions on distributing the GPL binaries
themselves. Falak never bundles FFmpeg: Falak's MIT license covers Falak's
own code, and each user downloads FFmpeg directly from the upstream release
page at first use, which keeps that separation. If you ever want to bundle
FFmpeg (or any GPL library) inside the installer instead, get legal advice
first — that changes the licensing picture.

## Libraries embedded in the WASM modules (gen2brain/avif, heic, webp)

- **gen2brain/avif** embeds **libavif** v1.4.2 with **aom** v3.14.1 and
  **dav1d** v1.5.3 (plus **libyuv**, per its build Makefile), compiled to
  WASM. Licenses: libavif, aom and dav1d are permissive BSD-style
  licenses (AOMedia / Joe Drago / VideoLAN); libyuv is a permissive
  BSD-style license. Full texts: `THIRD_PARTY_LICENSES.txt`.
- **gen2brain/webp** embeds **libwebp**, license BSD-3
  (Copyright (c) 2010, Google Inc.). Full text: `THIRD_PARTY_LICENSES.txt`.
- **gen2brain/heic** embeds the **pure-Rust `heic` crate** v0.1.6 and its
  Rust dependencies (per Cargo.lock). It contains **no C libraries** and
  **no LGPL code**.

**LGPL note:** the LGPL-3.0 libraries **libheif** and **libde265** are NOT
embedded; the gen2brain modules would only load them through their optional
dynamic-library path (via purego), which Falak builds out with the
`nodynamic` tag. For completeness: Falak's full source and build
instructions are public in this repository, so any embedded library can be
replaced and the app rebuilt from source.
