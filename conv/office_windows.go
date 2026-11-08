package conv

import "golang.org/x/sys/windows/registry"

func init() {
	exe := map[string]string{"word": "WINWORD.EXE", "excel": "EXCEL.EXE", "powerpoint": "POWERPNT.EXE"}
	msOfficeHas = func(app string) bool {
		name := exe[app]
		if name == "" {
			return false
		}
		for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			k, err := registry.OpenKey(root, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`+name, registry.QUERY_VALUE)
			if err == nil {
				k.Close()
				return true
			}
		}
		return false
	}
}
