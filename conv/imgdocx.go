package conv

// An image (JPG or PNG) converts to DOCX: a Word file that holds the
// picture, page-fitted. The file is written directly — Open Packaging (a
// ZIP holding document.xml and the picture) — so no office suite is needed.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// A4 page (11906×16838 twips) minus the 1440-twip margins, in EMU
// (635 EMU per twip, 914400 per inch).
const (
	docxPageWEMU = int64(9026 * 635)
	docxPageHEMU = int64(13958 * 635)
	emuPerPixel  = 9525 // one picture pixel at 96 dpi
)

func writeImageDocx(w io.Writer, img image.Image, srcPath string) error {
	media, ext, ctype := docxImageMedia(img, srcPath)
	b := img.Bounds()
	cx, cy := int64(b.Dx())*emuPerPixel, int64(b.Dy())*emuPerPixel
	// fit the page, never upscaling beyond the natural size
	if cx > 0 && cy > 0 {
		if s := min(docxPageWEMU*100/cx, docxPageHEMU*100/cy, 100); s < 100 {
			cx, cy = cx*s/100, cy*s/100
		}
	}
	z := zip.NewWriter(w)
	put := func(name string, body []byte) error {
		f, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(body)
		return err
	}
	const hdr = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
	if err := put("[Content_Types].xml", []byte(hdr+`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`+
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`+
		`<Default Extension="xml" ContentType="application/xml"/>`+
		`<Default Extension="`+ext+`" ContentType="`+ctype+`"/>`+
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`+
		`</Types>`)); err != nil {
		return err
	}
	if err := put("_rels/.rels", []byte(hdr+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>`+
		`</Relationships>`)); err != nil {
		return err
	}
	if err := put("word/_rels/document.xml.rels", []byte(hdr+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.`+ext+`"/>`+
		`</Relationships>`)); err != nil {
		return err
	}
	doc := hdr + fmt.Sprintf(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="1" name="Picture"/><wp:cNvGraphicFramePr/><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="1" name="Picture"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="rId1"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p><w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`, cx, cy, cx, cy)
	if err := put("word/document.xml", []byte(doc)); err != nil {
		return err
	}
	f, err := z.Create("media/image1." + ext)
	if err != nil {
		return err
	}
	if _, err := f.Write(media); err != nil {
		return err
	}
	return z.Close()
}

// docxImageMedia returns the picture bytes to embed (the original file's
// bytes when they can be used untouched), plus its extension and type.
func docxImageMedia(img image.Image, srcPath string) (data []byte, ext, ctype string) {
	if raw, err := os.ReadFile(srcPath); err == nil {
		switch Kind(srcPath) {
		case "JPG":
			if jpegOrientation(raw) <= 1 {
				return raw, "jpeg", "image/jpeg"
			}
		case "PNG":
			return raw, "png", "image/png"
		}
	}
	var buf bytes.Buffer
	if hasAlpha(img) && Kind(srcPath) != "JPG" {
		png.Encode(&buf, img)
		return buf.Bytes(), "png", "image/png"
	}
	jpeg.Encode(&buf, flatten(img), &jpeg.Options{Quality: 95})
	return buf.Bytes(), "jpeg", "image/jpeg"
}
