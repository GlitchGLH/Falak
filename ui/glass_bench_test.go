package ui

import (
	"image"
	"testing"
)

// BenchmarkGlassBaseDialog measures one glass rebuild at the compress
// dialog's size (428x384 at 125%): capture plus blur plus compositing.
func BenchmarkGlassBaseDialog(b *testing.B) {
	bg := image.NewRGBA(image.Rect(0, 0, 428, 384))
	for i := range bg.Pix {
		bg.Pix[i] = 128
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		glassBase(428, 384, 384, 340, 22, 32, 1.25, bg, false)
	}
}

// BenchmarkGlassBasePanel measures one glass rebuild at the progress
// panel's size (471x173 at 125%).
func BenchmarkGlassBasePanel(b *testing.B) {
	bg := image.NewRGBA(image.Rect(0, 0, 471, 173))
	for i := range bg.Pix {
		bg.Pix[i] = 128
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		glassBase(471, 173, 427, 129, 22, 32, 1.25, bg, false)
	}
}
