package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"raha/conv"
)

func main() {
	conv.FFmpegOverride = "/usr/bin/ffmpeg"
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(time.Second); cancel() }()
	t := time.Now()
	dst, err := conv.Run(ctx, os.Args[1], os.Args[2], nil)
	fmt.Printf("returned after %v: dst=%q err=%v\n", time.Since(t).Round(time.Millisecond), dst, err)
}
