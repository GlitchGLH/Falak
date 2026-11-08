package main

import (
	"bytes"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	"github.com/gen2brain/webp"
	"image"
	"image/color"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range img.Pix {
		img.Pix[i] = byte(i)
	}
	img.Set(1, 1, color.White)
	var b bytes.Buffer
	fmt.Println(webp.Encode(&b, img, webp.Options{Quality: 80}), b.Len())
	b.Reset()
	fmt.Println(avif.Encode(&b, img, avif.Options{Quality: 60}), b.Len())
	_ = gg.NewContext(10, 10)
}
