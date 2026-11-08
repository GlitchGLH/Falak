package ui

import "testing"

// TestShapeHamza: the hamza (U+0621) never joins. In "shin yeh hamza" the
// yeh takes its final form (it does not reach for the hamza) and the hamza
// stays, isolated - it used to become a NUL that drew nothing.
func TestShapeHamza(t *testing.T) {
	got := shapeArabic([]rune{0x0634, 0x064A, 0x0621})
	want := []rune{0xFEB7, 0xFEF2, 0xFE80} // shin initial, yeh final, hamza isolated
	if string(got) != string(want) {
		t.Fatalf("shapeArabic = %U, want %U", got, want)
	}
	for _, c := range got {
		if c == 0 {
			t.Fatal("NUL in the shaped text")
		}
	}
}

// TestShapeUnchanged: ordinary joining and lam-alef stay as before.
func TestShapeUnchanged(t *testing.T) {
	for _, c := range []struct {
		in, want []rune
	}{
		{[]rune{0x0644, 0x0627}, []rune{0xFEFB}},                         // lam alef: isolated ligature
		{[]rune{0x0628, 0x0644, 0x0627}, []rune{0xFE91, 0xFEFC}},         // beh, lam alef: final ligature
		{[]rune{0x0628, 0x064A, 0x062A}, []rune{0xFE91, 0xFEF4, 0xFE96}}, // beh yeh teh: initial, medial, final
		{[]rune{0x062F, 0x0627, 0x0631}, []rune{0xFEA9, 0xFE8D, 0xFEAD}}, // dal alef reh: none joins forward
	} {
		if got := shapeArabic(c.in); string(got) != string(c.want) {
			t.Errorf("shapeArabic(%U) = %U, want %U", c.in, got, c.want)
		}
	}
}
