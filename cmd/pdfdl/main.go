package main

import (
	"fmt"
	"os"

	"raha/conv"
)

func main() {
	conv.ToolsDir = os.Args[1]
	err := conv.EnsurePDFium(func(f float64) {})
	fmt.Println("err:", err, "have:", conv.HavePDFium(), conv.PDFiumPath())
}
