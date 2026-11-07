//go:build windows

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Falak copies itself to %LOCALAPPDATA%\Programs\Falak\Falak.exe and runs
// from there. A file opened straight from a download preview may live in a
// temporary folder that is cleaned up while it runs, after which Windows
// cannot start the administrator helper or the "start with Windows" entry.

func installDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Falak")
}

func installedExe() string { return filepath.Join(installDir(), "Falak.exe") }

// ensureInstalled returns true if this process should exit because the
// installed copy was started instead.
func ensureInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	target := installedExe()
	if strings.EqualFold(filepath.Clean(exe), filepath.Clean(target)) {
		return false
	}
	logf("running from %s; installing to %s", exe, target)
	if err := os.MkdirAll(installDir(), 0o755); err != nil {
		logf("install failed: %v", err)
		return false
	}
	if err := copyFile(exe, target); err != nil {
		logf("install failed: %v", err)
		return false
	}
	// Start the installed copy at the normal level (through File Explorer).
	if mutexHandle != 0 {
		pCloseHandle.Call(mutexHandle)
		mutexHandle = 0
	}
	r, _, _ := pShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16("explorer.exe"))),
		uintptr(unsafe.Pointer(utf16(`"`+target+`"`))), 0, 1)
	if r <= 32 {
		logf("could not start installed copy (code %d); continuing from here", r)
		mutexHandle, _, _ = pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16(`Local\FalakFileWheel`))))
		return false
	}
	return true
}

// runningFromInstall reports whether this process is the installed copy.
func runningFromInstall() bool {
	exe, err := os.Executable()
	return err == nil && strings.EqualFold(filepath.Clean(exe), filepath.Clean(installedExe()))
}

// replaceRunningCopy is used when a downloaded (newer) Falak starts while
// another copy runs: that copy is asked to close (it stops its conversions
// cleanly); a copy too old to understand the request is ended. Returns true
// once this process holds the single-instance lock.
func replaceRunningCopy() bool {
	if mutexHandle != 0 {
		pCloseHandle.Call(mutexHandle)
		mutexHandle = 0
	}
	h := findMainWindow()
	var pid uint32
	if h != 0 {
		pGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&pid)))
		pPostMessageW.Call(h, wmQuitUpdate, 0, 0)
	}
	take := func() bool {
		m, _, e := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16(`Local\FalakFileWheel`))))
		if m != 0 && e != windows.ERROR_ALREADY_EXISTS {
			mutexHandle = m
			return true
		}
		pCloseHandle.Call(m)
		return false
	}
	for i := 0; i < 60; i++ { // up to 6 s
		time.Sleep(100 * time.Millisecond)
		if take() {
			return true
		}
	}
	if pid != 0 {
		if p, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid); err == nil {
			logf("the running copy did not close; ending it")
			windows.TerminateProcess(p, 0)
			windows.WaitForSingleObject(p, 3000)
			windows.CloseHandle(p)
		}
	}
	return take()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		// still in use (e.g. by the key helper): a running program can be
		// renamed, so move it aside and put the new copy in its place
		old := dst + ".old"
		os.Remove(old)
		if err := os.Rename(dst, old); err != nil {
			os.Remove(tmp)
			return err
		}
	}
	return os.Rename(tmp, dst)
}

// cleanOldCopies removes copies that were moved aside during an update.
func cleanOldCopies() { os.Remove(installedExe() + ".old") }

const helperTask = `Falak\KeysHelper`

func hiddenCmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return c
}

// registerHelperTask is run by the elevated helper: it creates a scheduled
// task with highest privileges so later starts need no confirmation.
func registerHelperTask() {
	out, err := hiddenCmd("schtasks", "/Create", "/F", "/TN", helperTask,
		"/TR", `"`+installedExe()+`" --keys`, "/SC", "ONCE", "/ST", "00:00", "/RL", "HIGHEST").CombinedOutput()
	logf("register helper task: err=%v %s", err, strings.TrimSpace(string(out)))
}

// startHelperTask starts the helper through the scheduled task (no prompt).
func startHelperTask() bool {
	out, err := hiddenCmd("schtasks", "/Run", "/TN", helperTask).CombinedOutput()
	logf("start helper task: err=%v %s", err, strings.TrimSpace(string(out)))
	return err == nil
}
