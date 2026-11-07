package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	approvedPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	layersPath   = `Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`
)

func startupEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValueName)
	return err == nil
}

// setStartup always points Windows at the installed copy, which runs at the
// normal level. (Windows silently skips startup entries that need
// administrator rights, e.g. a copy marked "Run as administrator".)
func setStartup(on bool) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.ALL_ACCESS)
	if err != nil {
		logf("startup: cannot open Run key: %v", err)
		return
	}
	defer k.Close()
	if !on {
		k.DeleteValue(runValueName)
		logf("startup: turned off")
		return
	}
	want := `"` + installedExe() + `" --startup`
	k.SetStringValue(runValueName, want)
	// Undo a "disabled" mark from Task Manager / Settings > Startup apps.
	if a, err := registry.OpenKey(registry.CURRENT_USER, approvedPath, registry.ALL_ACCESS); err == nil {
		if v, _, err := a.GetBinaryValue(runValueName); err == nil && len(v) > 0 && v[0]&1 == 1 {
			a.SetBinaryValue(runValueName, []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			logf("startup: was disabled in Startup apps; enabled again")
		}
		a.Close()
	}
	// Remove "Run as administrator" marks on Falak's own files.
	if l, err := registry.OpenKey(registry.CURRENT_USER, layersPath, registry.ALL_ACCESS); err == nil {
		names, _ := l.ReadValueNames(-1)
		for _, n := range names {
			if !strings.HasSuffix(strings.ToLower(n), `\falak.exe`) {
				continue
			}
			v, _, err := l.GetStringValue(n)
			if err != nil || !strings.Contains(strings.ToUpper(v), "RUNASADMIN") {
				continue
			}
			rest := strings.Fields(strings.ReplaceAll(strings.ToUpper(v), "RUNASADMIN", ""))
			if len(rest) == 0 || (len(rest) == 1 && rest[0] == "~") {
				l.DeleteValue(n)
			} else {
				l.SetStringValue(n, strings.Join(rest, " "))
			}
			logf("startup: removed 'run as administrator' mark from %s", n)
		}
		l.Close()
	}
	logf("startup: on -> %s", want)
}

// refreshStartup keeps an enabled startup entry pointing at the current
// installed copy (older builds could have pointed elsewhere).
func refreshStartup() {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	v, _, err := k.GetStringValue(runValueName)
	k.Close()
	if err != nil {
		return
	}
	logf("startup entry: %s", v)
	setStartup(true)
}
