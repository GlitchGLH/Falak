package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestVisualMixed(t *testing.T) {
	for _, c := range []string{
		"سلام World",
		"اضغط Alt أو Ctrl أو Tab",
		"اسحب ملفًا: تُفتح العجلة حوله.",
		"خفض 50% من الجودة",
		"الضغط إلى حجم محدد",
	} {
		if got := Visual(c); !utf8.ValidString(got) {
			t.Errorf("Visual(%q) invalid utf8", c)
		}
	}
	// a Latin word after Arabic keeps its leading space
	if got := Visual("سلام World"); !strings.HasPrefix(got, "World ") {
		t.Errorf("Arabic-then-Latin order: %q", got)
	}
	// spaces survive around every Latin word
	if got := Visual("اضغط Alt أو Ctrl أو Tab"); strings.Count(got, " ") != 5 {
		t.Errorf("spaces lost around Latin words: %q", got)
	}
	// Latin islands read left to right: Ctrl lands left of Alt
	got := Visual("اضغط Alt أو Ctrl")
	if strings.Index(got, "Ctrl") > strings.Index(got, "Alt") {
		t.Errorf("island order wrong: %q", got)
	}
	// a percent sign stays with its number
	if got := Visual("خفض 50% من الجودة"); !strings.Contains(got, "50%") {
		t.Errorf("number mangled: %q", got)
	}
	// plain Latin passes through untouched
	if got := Visual("Converting to MP3"); got != "Converting to MP3" {
		t.Errorf("latin changed: %q", got)
	}
}

// The credit line under the menu title must survive Visual() unchanged.
func TestVisualCredit(t *testing.T) {
	if got := Visual("1.0 · By UncleGlitch"); !strings.Contains(got, "By UncleGlitch") {
		t.Errorf("credit mangled: %q", got)
	}
}
