package conv

// Plain text files (.txt) and the ASS/SSA subtitle format.
//
// A .txt converts to documents (DOCX, RTF, HTML, and PDF through the
// installed office suite) and, through that PDF, to one JPG or PNG per
// page. It also converts to SRT and VTT: when the file holds timed
// subtitles they are kept, otherwise every non-empty line becomes one cue
// timed by its length. Text is read as UTF-8 / UTF-16 (with or without
// BOM) or, failing that, Windows-1256, the usual Arabic encoding of older
// Windows files.

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// ---------- decoding ----------

func decodeText(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF:
		return string(data[3:])
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		return decodeUTF16(data[2:], false)
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		return decodeUTF16(data[2:], true)
	case utf8.Valid(data):
		return string(data)
	}
	if s, err := charmap.Windows1256.NewDecoder().Bytes(data); err == nil {
		return string(s)
	}
	return strings.ToValidUTF8(string(data), "\uFFFD")
}

func decodeUTF16(b []byte, big bool) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		if big {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i+1])<<8 | uint16(b[2*i])
		}
	}
	return string(utf16.Decode(u))
}

// ---------- ASS / SSA ----------

func isASS(text string) bool {
	low := strings.ToLower(text)
	return strings.Contains(low, "[events]") && strings.Contains(low, "dialogue:")
}

var assTagRe = strings.NewReplacer(`\N`, "\n", `\n`, "\n", `\h`, " ")

// parseASS reads the Dialogue lines of the [Events] section, using its
// Format line to find the Start, End and Text fields.
func parseASS(text string) []cue {
	var cues []cue
	start, end, txt, nf := 1, 2, 9, 10
	inEvents := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		switch {
		case strings.HasPrefix(line, "["):
			inEvents = low == "[events]"
		case inEvents && strings.HasPrefix(low, "format:"):
			fields := strings.Split(line[len("format:"):], ",")
			nf = len(fields)
			for i, f := range fields {
				switch strings.ToLower(strings.TrimSpace(f)) {
				case "start":
					start = i
				case "end":
					end = i
				case "text":
					txt = i
				}
			}
		case inEvents && strings.HasPrefix(low, "dialogue:"):
			parts := strings.SplitN(line[len("dialogue:"):], ",", nf)
			if len(parts) <= max(start, end, txt) {
				continue
			}
			t0, ok0 := assTime(parts[start])
			t1, ok1 := assTime(parts[end])
			if !ok0 || !ok1 {
				continue
			}
			body := stripASSTags(parts[txt])
			var lines []string
			for _, l := range strings.Split(assTagRe.Replace(body), "\n") {
				if l = strings.TrimSpace(l); l != "" {
					lines = append(lines, l)
				}
			}
			if len(lines) == 0 {
				continue
			}
			cues = append(cues, cue{timing: fmtMS(t0, '.') + " --> " + fmtMS(t1, '.'), lines: lines})
		}
	}
	return cues
}

// stripASSTags removes override blocks such as {\i1} or {\pos(10,20)}.
func stripASSTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '{':
			depth++
		case r == '}' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// assTime parses H:MM:SS.cc into milliseconds.
func assTime(s string) (int, bool) {
	s = strings.TrimSpace(s)
	p := strings.Split(s, ":")
	if len(p) != 3 {
		return 0, false
	}
	h, e1 := strconv.Atoi(p[0])
	m, e2 := strconv.Atoi(p[1])
	sec, e3 := strconv.ParseFloat(p[2], 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return 0, false
	}
	return (h*3600+m*60)*1000 + int(sec*1000+0.5), true
}

// srtTime parses HH:MM:SS,mmm (or with a dot) into milliseconds.
func srtTime(s string) int {
	s = strings.Replace(strings.TrimSpace(s), ",", ".", 1)
	ms, _ := assTime(s)
	return ms
}

func fmtMS(ms int, sep byte) string {
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", ms/3600000, ms/60000%60, ms/1000%60, sep, ms%1000)
}

func writeASS(b *strings.Builder, cues []cue) {
	b.WriteString("[Script Info]\nScriptType: v4.00+\nWrapStyle: 0\nScaledBorderAndShadow: yes\nPlayResX: 1920\nPlayResY: 1080\n\n")
	b.WriteString("[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	b.WriteString("Style: Default,Arial,64,&H00FFFFFF,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,3,1,2,60,60,50,1\n\n")
	b.WriteString("[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	at := func(ms int) string {
		cs := (ms + 5) / 10
		return fmt.Sprintf("%d:%02d:%02d.%02d", cs/360000, cs/6000%60, cs/100%60, cs%100)
	}
	for _, c := range cues {
		parts := strings.SplitN(normTiming(c.timing, false), "-->", 2)
		var ls []string
		for _, l := range c.lines {
			ls = append(ls, strings.TrimSpace(reTags.ReplaceAllString(l, "")))
		}
		fmt.Fprintf(b, "Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", at(srtTime(parts[0])), at(srtTime(parts[1])), strings.Join(ls, `\N`))
	}
}

// ---------- plain text ----------

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := strings.ReplaceAll(strings.ReplaceAll(decodeText(data), "\r\n", "\n"), "\r", "\n")
	return strings.Split(strings.TrimRight(t, "\n"), "\n"), nil
}

func textItems(path string) []Item {
	ok := msOfficeHas("word") || libreOffice() != ""
	out := []Item{{ID: "to:PDF", Label: "PDF", Enabled: ok}, {ID: "to:JPG", Label: "JPG", Enabled: ok},
		{ID: "to:PNG", Label: "PNG", Enabled: ok}, {ID: "to:SRT", Label: "SRT", Enabled: true},
		{ID: "to:VTT", Label: "VTT", Enabled: true}, {ID: "to:DOCX", Label: "DOCX", Enabled: true},
		{ID: "to:RTF", Label: "RTF", Enabled: true}, {ID: "to:HTML", Label: "HTML", Enabled: true}}
	// ASS stays an extra for files that hold timed subtitles (only small
	// files are looked into)
	if fi, err := os.Stat(path); err != nil || fi.Size() > 8<<20 {
		return out
	}
	if data, err := os.ReadFile(path); err == nil && len(parseSubs(data)) > 0 {
		out = append(out, Item{ID: "to:ASS", Label: "ASS", Enabled: true})
	}
	return out
}

func runText(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	t := strings.TrimPrefix(id, "to:")
	switch t {
	case "SRT", "VTT":
		return textToSubs(path, t)
	case "ASS":
		return runSubtitle(path, id)
	}
	lines, err := readLines(path)
	if err != nil {
		return "", err
	}
	switch t {
	case "DOCX":
		dst := uniquePath(path, "", ".docx")
		return writeAtomic(dst, func(w io.Writer) error { return writeDocx(w, lines) })
	case "RTF":
		dst := uniquePath(path, "", ".rtf")
		return writeAtomic(dst, func(w io.Writer) error { _, err := io.WriteString(w, makeRTF(lines)); return err })
	case "HTML":
		dst := uniquePath(path, "", ".html")
		return writeAtomic(dst, func(w io.Writer) error { _, err := io.WriteString(w, makeHTML(path, lines)); return err })
	case "JPG", "PNG":
		// the text is laid out into a PDF first, then each page becomes
		// one image (the same naming as PDF to JPG/PNG)
		pdf, err := textToPDF(ctx, path, lines, progress)
		if err != nil {
			return "", err
		}
		defer dropTemp(pdf)
		return pdfToImages(ctx, path, pdf, t, progress)
	case "PDF":
		dst := uniquePath(path, "", ".pdf")
		pdf, err := textToPDF(ctx, path, lines, progress)
		if err != nil {
			return "", err
		}
		return finishTemp(pdf, dst)
	}
	return "", fmt.Errorf("unsupported %s", t)
}

// textToPDF lays the text out into a PDF through the installed office
// suite (Word or LibreOffice, which shape Arabic correctly) and returns
// the temporary PDF's path; the caller removes it.
func textToPDF(ctx context.Context, path string, lines []string, progress func(float64)) (string, error) {
	stop := estimate(progress, 0.02, 0.99, 5+fileMB(path)*0.8)
	defer stop()
	docx := tempFor(path) + ".docx"
	f, err := os.Create(docx)
	if err != nil {
		return "", err
	}
	err = writeDocx(f, lines)
	f.Close()
	defer dropTemp(docx)
	if err != nil {
		return "", err
	}
	tmp := tempFor(docx)
	if msOfficeHas("word") {
		err = officeMS(ctx, "word", docx, tmp, msFormat["word"]["PDF"])
	} else if lo := libreOffice(); lo != "" {
		err = officeLO(ctx, lo, docx, tmp, loFormat["PDF"])
	} else {
		err = errors.New(tr("This needs Microsoft Office or LibreOffice installed", "هذا يحتاج تثبيت Microsoft Office أو LibreOffice"))
	}
	if err == nil {
		if _, e := os.Stat(tmp); e != nil {
			err = errors.New(tr("The office program didn't produce a file", "لم يُنتج برنامج Office ملفًا"))
		}
	}
	if err != nil {
		dropTemp(tmp)
		return "", err
	}
	return tmp, nil
}

// pdfToImages renders a PDF's pages as JPG or PNG images, with the same
// naming as PDF to JPG/PNG: one image per page in a new folder, or a
// single file for a single page.
func pdfToImages(ctx context.Context, path, pdf, t string, progress func(float64)) (string, error) {
	doc, err := openPDF(pdf)
	if err != nil {
		return "", err
	}
	defer doc.Close()
	n := doc.Pages()
	if n < 1 {
		return "", errors.New(tr("The PDF has no pages", "ملف PDF لا يحتوي على صفحات"))
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	page := func(i int, dst string, lo, hi float64) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		stopR := estimate(progress, lo, lo+(hi-lo)*0.3, 0.6)
		img, err := doc.Render(i, PDFDPI)
		stopR()
		if err != nil {
			return "", err
		}
		return writeAtomic(dst, func(w io.Writer) error {
			if progress == nil {
				return EncodeBest(w, img, t, true)
			}
			stop := estimate(progress, lo+(hi-lo)*0.3, lo+(hi-lo)*0.99, encodeSeconds(img, t, true))
			defer stop()
			return EncodeBest(w, img, t, true)
		})
	}
	if n == 1 {
		dst := uniquePath(path, "", extFor(t))
		return page(0, dst, 0, 1)
	}
	root := uniqueDir(filepath.Dir(path), base+" - "+t)
	tmp := tempFor(root)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	width := len(fmt.Sprint(n))
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s - %0*d%s", base, width, i+1, extFor(t))
		lo, hi := float64(i)/float64(n), float64(i+1)/float64(n)
		if _, err := page(i, filepath.Join(tmp, name), lo, hi); err != nil {
			dropTemp(tmp)
			return "", err
		}
		if progress != nil {
			progress(float64(i+1) / float64(n))
		}
	}
	return finishTemp(tmp, root)
}

// textToSubs makes SRT or VTT from a plain text file: when it holds timed
// subtitles they are kept, otherwise every non-empty line becomes one cue,
// timed from zero by its length.
func textToSubs(path, t string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	cues := parseSubs(data)
	if len(cues) == 0 {
		lines, _ := readLines(path)
		cues = lineCues(lines)
	}
	if len(cues) == 0 {
		return "", errors.New(tr("No subtitles found in the file", "لم يُعثر على ترجمات داخل الملف"))
	}
	var b strings.Builder
	if t == "VTT" {
		b.WriteString("WEBVTT\n\n")
		for _, c := range cues {
			fmt.Fprintf(&b, "%s\n%s\n\n", normTiming(c.timing, false), strings.Join(c.lines, "\n"))
		}
	} else {
		for i, c := range cues {
			fmt.Fprintf(&b, "%d\n%s\n%s\n\n", i+1, normTiming(c.timing, true), strings.Join(c.lines, "\n"))
		}
	}
	dst := uniquePath(path, "", "."+strings.ToLower(t))
	return writeAtomic(dst, func(w io.Writer) error { _, err := io.WriteString(w, b.String()); return err })
}

// lineCues times the non-empty lines of a plain text file from zero: each
// line shows for 1 s plus 60 ms per character, between 1.5 s and 7 s.
func lineCues(lines []string) []cue {
	var out []cue
	at := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		d := 1000 + 60*len([]rune(l))
		if d < 1500 {
			d = 1500
		}
		if d > 7000 {
			d = 7000
		}
		out = append(out, cue{timing: fmtMS(at, ',') + " --> " + fmtMS(at+d, ','), lines: []string{l}})
		at += d
	}
	return out
}

// rtlLine reports whether a line starts (in its first strong letter) with
// a right-to-left script.
func rtlLine(s string) bool {
	for _, r := range s {
		if isRTLRune(r) {
			return true
		}
		if unicode.IsLetter(r) {
			return false
		}
	}
	return false
}

func isRTLRune(r rune) bool {
	return (r >= 0x0590 && r <= 0x08FF) || (r >= 0xFB1D && r <= 0xFDFF) || (r >= 0xFE70 && r <= 0xFEFF)
}

func xmlEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '\t' || r >= 0x20 && r != 0xFFFE && r != 0xFFFF:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeDocx(w io.Writer, lines []string) error {
	z := zip.NewWriter(w)
	put := func(name, body string) error {
		f, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}
	const hdr = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
	if err := put("[Content_Types].xml", hdr+`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`+
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`+
		`<Default Extension="xml" ContentType="application/xml"/>`+
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`+
		`</Types>`); err != nil {
		return err
	}
	if err := put("_rels/.rels", hdr+`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>`+
		`</Relationships>`); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(hdr + `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, l := range lines {
		rpr := `<w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Arial"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr>`
		if rtlLine(l) {
			b.WriteString(`<w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr>`)
			rpr = strings.Replace(rpr, "<w:sz ", "<w:rtl/><w:sz ", 1)
		} else {
			b.WriteString(`<w:p>`)
		}
		if l != "" {
			b.WriteString(`<w:r>` + rpr + `<w:t xml:space="preserve">` + xmlEsc(l) + `</w:t></w:r>`)
		}
		b.WriteString(`</w:p>`)
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr></w:body></w:document>`)
	if err := put("word/document.xml", b.String()); err != nil {
		return err
	}
	return z.Close()
}

func makeRTF(lines []string) string {
	var b strings.Builder
	b.WriteString(`{\rtf1\ansi\ansicpg1252\deff0{\fonttbl{\f0\fswiss\fcharset0 Calibri;}{\f1\fnil\fcharset178 Arial;}}` + "\n" + `\uc1\f0\fs24 `)
	for _, l := range lines {
		if rtlLine(l) {
			b.WriteString(`\pard\rtlpar\qr\f1\rtlch `)
		} else {
			b.WriteString(`\pard\ltrpar\ql\f0\ltrch `)
		}
		for _, r := range l {
			switch {
			case r == '\\' || r == '{' || r == '}':
				b.WriteString(`\` + string(r))
			case r == '\t':
				b.WriteString(`\tab `)
			case r < 0x80:
				b.WriteRune(r)
			case r < 0x10000:
				fmt.Fprintf(&b, `\u%d?`, int16(r))
			default:
				for _, u := range utf16.Encode([]rune{r}) {
					fmt.Fprintf(&b, `\u%d?`, int16(u))
				}
			}
		}
		b.WriteString("\\par\n")
	}
	b.WriteString("}")
	return b.String()
}

func makeHTML(path string, lines []string) string {
	var b strings.Builder
	b0 := baseName(path)
	title := b0[:len(b0)-len(filepath.Ext(b0))]
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<title>" + html.EscapeString(title) + "</title>\n")
	b.WriteString("<style>body{font-family:'Segoe UI',Tahoma,Arial,sans-serif;font-size:16px;line-height:1.6;max-width:820px;margin:40px auto;padding:0 20px}p{margin:0 0 .4em;white-space:pre-wrap}</style>\n</head>\n<body>\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			b.WriteString("<p>&nbsp;</p>\n")
			continue
		}
		b.WriteString("<p dir=\"auto\">" + html.EscapeString(l) + "</p>\n")
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}
