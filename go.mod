module raha

go 1.25.0

// GODEBUG defaults stay those of Go 1.24 (what Falak 1.0.0 was built
// with), even though some dependencies declare go 1.25.0.
godebug default=go1.24

require (
	github.com/bodgit/sevenzip v1.6.0
	github.com/fogleman/gg v1.3.1-0.20210928143535-8febc0f526ad
	github.com/gen2brain/avif v0.6.1-0.20260814210334-6cb8f35756be
	github.com/gen2brain/heic v0.7.2
	github.com/gen2brain/webp v0.6.5-0.20260815184508-9217dec533b1
	github.com/nwaples/rardecode/v2 v2.4.6
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/ulikunitz/xz v0.5.12
	golang.org/x/image v0.24.0
	golang.org/x/sys v0.44.0
	golang.org/x/text v0.22.0
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/bodgit/plumbing v1.3.0 // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/tetratelabs/wazero v1.12.1-0.20261019203559-ef2dd2f85376 // indirect
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/net v0.35.0 // indirect
)

// These are the exact versions Falak 1.0.0 is built from. Some dependencies
// ask for newer ones (avif/heic: x/sys v0.44.0, purego v0.10.1; sevenzip:
// brotli v1.1.1), which minimal version selection would otherwise pick.
replace (
	github.com/andybalholm/brotli => github.com/andybalholm/brotli v1.1.0
	github.com/ebitengine/purego => github.com/ebitengine/purego v0.8.4
	golang.org/x/sys => golang.org/x/sys v0.30.0
)
