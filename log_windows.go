//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

var (
	logMu   sync.Mutex
	logFile *os.File
	logPath string
	logN    int
)

// openLog writes diagnostics to %LOCALAPPDATA%\Falak\falak.log. If the app
// ever crashes, the Go runtime also writes the crash report there.
func openLog() {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Falak")
	os.MkdirAll(dir, 0o755)
	logPath = filepath.Join(dir, "falak.log")
	if st, err := os.Stat(logPath); err == nil && st.Size() > 2<<20 {
		os.Rename(logPath, logPath+".old")
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	logFile = f
	debug.SetCrashOutput(f, debug.CrashOptions{})
}

func logf(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil || logN > 5000 {
		return
	}
	logN++
	fmt.Fprintf(logFile, "%s  %s\r\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
}
