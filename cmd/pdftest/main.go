// Command pdftest converts files with conv.Run (Windows test helper).
package main

import (
	"context"
	"fmt"
	"os"

	"raha/conv"
)

func main() {
	conv.PDFiumOverride = os.Getenv("PDFIUM")
	conv.FFmpegOverride = os.Getenv("FFMPEG")
	conv.ResvgOverride = os.Getenv("RESVG")
	for i := 1; i+1 < len(os.Args); i += 2 {
		dst, err := conv.Run(context.Background(), os.Args[i], os.Args[i+1], nil)
		fmt.Println(os.Args[i], os.Args[i+1], "->", dst, err)
	}
}
