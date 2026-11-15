//go:build windows

package main

import "testing"

// TestSplitCountText: the counter's English and the owner's approved
// Arabic forms by number.
func TestSplitCountText(t *testing.T) {
	cases := []struct {
		n      int
		en, ar string
	}{
		{0, "0 split points", "لا توجد نقاط تقسيم"},
		{1, "1 split point", "نقطة تقسيم واحدة"},
		{2, "2 split points", "نقطتا تقسيم"},
		{3, "3 split points", "3 نقاط تقسيم"},
		{10, "10 split points", "10 نقاط تقسيم"},
		{11, "11 split points", "11 نقطة تقسيم"},
		{25, "25 split points", "25 نقطة تقسيم"},
	}
	for _, c := range cases {
		if got := splitCountText(c.n, false); got != c.en {
			t.Errorf("%d in English: %q, want %q", c.n, got, c.en)
		}
		if got := splitCountText(c.n, true); got != c.ar {
			t.Errorf("%d in Arabic: %q, want %q", c.n, got, c.ar)
		}
	}
}
