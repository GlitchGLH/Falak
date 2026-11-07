//go:build windows

// Command wictest decodes files with Windows Imaging Component.
package main

import (
	"context"
	"fmt"
	"os"

	"raha/conv"
)

func main() {
	for _, p := range os.Args[1:] {
		img, err := conv.WICDecodeFile(p)
		if err != nil {
			fmt.Println(p, "error:", err)
			continue
		}
		b := img.Bounds()
		c := img.At(b.Dx()/2, b.Dy()/2)
		r, g, bl, a := c.RGBA()
		fmt.Printf("%s: %dx%d centre=(%d,%d,%d,%d)\n", p, b.Dx(), b.Dy(), r>>8, g>>8, bl>>8, a>>8)
	}
	for i := 1; i < len(os.Args); i++ {
		if len(os.Args[i]) > 4 && os.Args[i][len(os.Args[i])-4:] == ".NEF" {
			dst, err := conv.Run(context.Background(), os.Args[i], "to:JPG", nil)
			fmt.Println("RAW convert:", dst, err)
		}
	}
}
