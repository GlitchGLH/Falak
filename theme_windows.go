//go:build windows

package main

import "golang.org/x/sys/windows/registry"

// The app theme (light or dark) for the wheel, the progress window and the
// tray menu. It is the user's own choice in the tray menu, kept in the
// registry; the default is the light theme.

var darkTheme = loadDarkTheme()

func loadDarkTheme() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Falak`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Theme")
	return err == nil && v == "dark"
}

func darkMode() bool { return darkTheme }

func setDarkMode(on bool) {
	if k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Falak`, registry.ALL_ACCESS); err == nil {
		if on {
			k.SetStringValue("Theme", "dark")
		} else {
			k.SetStringValue("Theme", "light")
		}
		k.Close()
	}
	darkTheme = on
}
