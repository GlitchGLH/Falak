//go:build windows

package main

import "testing"

// TestNextFocus: Tab and the arrows move between Yes and No as the
// buttons are laid out (mirrored in Arabic); one button never moves.
func TestNextFocus(t *testing.T) {
	cases := []struct {
		focus, n int
		key      uintptr
		shift    bool
		rtl      bool
		want     int
	}{
		{0, 1, VK_TAB, false, false, 0},
		{0, 2, VK_TAB, false, false, 1},
		{1, 2, VK_TAB, false, false, 0},
		{0, 2, VK_TAB, true, false, 1},
		{0, 2, vkRight, false, false, 1},
		{1, 2, vkLeft, false, false, 0},
		{1, 2, vkRight, false, false, 0},
		{0, 2, vkLeft, false, true, 1}, // Arabic: No is left of Yes
		{1, 2, vkRight, false, true, 0},
		{0, 2, VK_ESCAPE, false, false, 0},
	}
	for _, c := range cases {
		if got := nextFocus(c.focus, c.n, c.key, c.shift, c.rtl); got != c.want {
			t.Errorf("nextFocus(%d, %d, %#x, shift %v, rtl %v) = %d, want %d", c.focus, c.n, c.key, c.shift, c.rtl, got, c.want)
		}
	}
}
