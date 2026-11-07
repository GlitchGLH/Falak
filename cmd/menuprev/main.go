// Command menuprev renders the tray menu (dark/light, English/Arabic) to
// /tmp/menu.png.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	xdraw "golang.org/x/image/draw"

	"raha/ui"
)

func main() {
	dj := "/usr/share/fonts/truetype/dejavu/"
	strong := ui.LoadFonts([]string{dj + "DejaVuSans-Bold.ttf"}, []string{dj + "DejaVuSans-Bold.ttf"})
	plain := ui.LoadFonts([]string{dj + "DejaVuSans.ttf"}, []string{dj + "DejaVuSans.ttf"})
	if len(os.Args) > 1 {
		in := "/usr/share/fonts/opentype/inter/"
		strong = ui.LoadFonts([]string{in + "Inter-Bold.otf"}, []string{in + "Inter-SemiBold.otf"})
		plain = ui.LoadFonts([]string{in + "Inter-Bold.otf"}, []string{in + "Inter-Regular.otf"})
	}
	var tiles []*image.RGBA
	for _, ar := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			T := func(e, a string) string {
				if ar {
					return a
				}
				return e
			}
			m := &ui.Menu{Scale: 1.5, Dark: dark, RTL: ar, Strong: strong, Plain: plain, Hover: 2,
				Title: T("Falak", "فلك"), Sub: "1.0 · By UncleGlitch",
				Logo: menuLogo(),
				Rows: []ui.MenuRow{
					{ID: "theme", Icon: "moon", Title: T("Dark theme", "الثيم الداكن"), Sub: T("On", "مفعّل"), SubBlue: true, Toggle: true, On: dark},
					{ID: "sound", Icon: "sound", Title: T("Wheel sounds", "أصوات العجلة"), Sub: T("Hover and drop sounds", "صوت عند التمرير والإفلات"), Toggle: true, On: true},
					{ID: "startup", Icon: "startup", Title: T("Start with Windows", "التشغيل مع بدء ويندوز"), Sub: T("Opens when you sign in", "يعمل عند تسجيل الدخول"), Toggle: true, On: true},
					{ID: "helper", Icon: "shield", Title: T("Key-reading helper", "مساعد قراءة الأزرار"), Sub: T("Administrator", "بصلاحيات المسؤول"), Toggle: true, On: false},
					{ID: "lang", Icon: "globe", Title: "العربية", Sub: T("Arabic interface", "واجهة عربية"), Toggle: true, On: ar},
					{ID: "help", Icon: "help", Title: T("How to use", "طريقة الاستخدام"), Footer: true},
					{ID: "quit", Icon: "power", Title: T("Quit", "إنهاء"), Footer: true},
				}}
			tiles = append(tiles, m.Render())
		}
	}
	w, h := tiles[0].Rect.Dx(), tiles[0].Rect.Dy()
	sheet := image.NewRGBA(image.Rect(0, 0, w*len(tiles), h))
	draw.Draw(sheet, sheet.Rect, &image.Uniform{color.RGBA{120, 130, 150, 255}}, image.Point{}, draw.Src)
	for i, t := range tiles {
		draw.Draw(sheet, image.Rect(w*i, 0, w*i+w, h), t, image.Point{}, draw.Over)
	}
	o, _ := os.Create("/tmp/menu.png")
	png.Encode(o, sheet)
	o.Close()
}

// menuLogo decodes the app icon and scales it to the header's logo size
// (26 logical px at the preview's scale of 1.5).
func menuLogo() *image.RGBA {
	f, err := os.Open("res/icon-1024.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	const px = 39
	dst := image.NewRGBA(image.Rect(0, 0, px, px))
	xdraw.CatmullRom.Scale(dst, dst.Rect, src, src.Bounds(), xdraw.Over, nil)
	return dst
}
