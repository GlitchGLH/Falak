package conv

// Arabic switches user-facing messages to Arabic (English is the default).
var Arabic bool

func tr(en, ar string) string {
	if Arabic {
		return ar
	}
	return en
}
