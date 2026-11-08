package main

import (
	"context"
	"fmt"
	"os"

	"raha/conv"
)

func main() {
	conv.ToolsDir = os.Args[1]

	big, err := conv.Run(context.Background(), `C:\m.png`, "to:PDF", nil)
	fmt.Println("png->pdf", big, err)
	out, err := conv.Run(context.Background(), big, "p:compress", nil)
	fmt.Println("compress", out, err)
	out, err = conv.Run(context.Background(), `C:\doc-rotated.pdf`, "p:compress", nil)
	fmt.Println("compress2", out, err)
	out, err = conv.Merge(context.Background(), []string{`C:\doc.pdf`, big, `C:\doc-rotated.pdf`}, "p:merge", nil)
	fmt.Println("merge", out, err)
}
