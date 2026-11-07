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
	start := time.Now()
	var last float64
	var lastAt time.Time
	n := 0
	_, err := conv.Run(context.Background(), os.Args[1], os.Args[2], func(f float64) { last, lastAt = f, time.Now(); n++ })
	end := time.Now()
	fmt.Printf("%s: %d updates, last %.1f%%, file ready %v after the last update (total %v) err=%v\n",
		os.Args[2], n, last*100, end.Sub(lastAt).Round(time.Millisecond), end.Sub(start).Round(time.Millisecond), err)
}
