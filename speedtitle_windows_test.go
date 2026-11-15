//go:build windows

package main

import "testing"

// TestSpeedTitle: the progress window's title names the speed, with the
// owner's approved Arabic.
func TestSpeedTitle(t *testing.T) {
	saved := arabic
	defer func() { arabic = saved }()
	for _, c := range []struct {
		ar    bool
		speed float64
		want  string
	}{
		{false, 0.5, "Changing the speed to 0.5×"},
		{false, 4, "Changing the speed to 4×"},
		{true, 0.25, "جار تغيير السرعة إلى 0.25×"},
		{true, 2, "جار تغيير السرعة إلى 2×"},
	} {
		arabic = c.ar
		if got := speedTitle(c.speed); got != c.want {
			t.Errorf("%v (arabic %v): %q, want %q", c.speed, c.ar, got, c.want)
		}
	}
}
