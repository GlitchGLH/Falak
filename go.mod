module raha

go 1.24.7

require (
	github.com/bodgit/sevenzip v1.6.0
	github.com/fogleman/gg v1.3.0
	github.com/gen2brain/avif v0.0.0
	github.com/gen2brain/heic v0.0.0
	github.com/gen2brain/webp v0.0.0
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/nwaples/rardecode/v2 v2.4.6
	github.com/srwiley/oksvg v0.0.0
	github.com/srwiley/rasterx v0.0.0
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
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/tetratelabs/wazero v1.12.0 // indirect
	go4.org v0.0.0-20200411211856-f5505b9728dd // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
)

replace (
	github.com/andybalholm/brotli => ../deps/brotli
	github.com/bodgit/plumbing => ../deps/plumbing
	github.com/bodgit/sevenzip => ../deps/sevenzip
	github.com/bodgit/windows => ../deps/bwindows
	github.com/davecgh/go-spew => ../deps/go-spew
	github.com/ebitengine/purego => ../deps/purego
	github.com/fogleman/gg => ../deps/gg
	github.com/gen2brain/avif => ../deps/avif
	github.com/gen2brain/heic => ../deps/heic
	github.com/gen2brain/webp => ../deps/webp
	github.com/golang/freetype => ../deps/freetype
	github.com/hashicorp/errwrap => ../deps/errwrap
	github.com/hashicorp/go-multierror => ../deps/multierror
	github.com/hashicorp/golang-lru/v2 => ../deps/golang-lru
	github.com/klauspost/compress => ../deps/compress
	github.com/nwaples/rardecode/v2 => ../deps/rardecode
	github.com/pierrec/lz4/v4 => ../deps/lz4
	github.com/pmezard/go-difflib => ../deps/go-difflib
	github.com/srwiley/oksvg => ../deps/oksvg
	github.com/srwiley/rasterx => ../deps/rasterx
	github.com/stretchr/objx => ../deps/objx
	github.com/stretchr/testify => ../deps/testify
	github.com/tetratelabs/wazero => ../deps/wazero
	github.com/ulikunitz/xz => ../deps/xz
	go4.org => ../deps/go4min
	golang.org/x/image => ../deps/image
	golang.org/x/net => ../deps/net
	golang.org/x/sync => ../deps/xsync
	golang.org/x/sys => ../deps/sys
	golang.org/x/text => ../deps/text
	gopkg.in/check.v1 => ../deps/stub-checkv1
	gopkg.in/yaml.v3 => ../deps/yaml3
)
