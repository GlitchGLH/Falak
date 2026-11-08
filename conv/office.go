package conv

// Word, Excel and PowerPoint documents are converted by the office suite
// installed on the PC: Microsoft Office (driven through its own automation,
// the most faithful result) or else LibreOffice.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var officeExts = map[string]string{
	".docx": "word", ".doc": "word", ".rtf": "word", ".odt": "word",
	".xlsx": "excel", ".xls": "excel", ".ods": "excel",
	".pptx": "powerpoint", ".ppt": "powerpoint", ".odp": "powerpoint",
}

var officeTargets = map[string][]string{
	"word":       {"PDF", "DOCX", "TXT", "RTF", "ODT"},
	"excel":      {"PDF", "XLSX", "CSV", "ODS"},
	"powerpoint": {"PDF", "PPTX", "PNG", "ODP"},
}

// MS Office save-format numbers, per application.
var msFormat = map[string]map[string]int{
	"word":       {"PDF": 17, "DOCX": 16, "TXT": 7, "RTF": 6, "ODT": 23},
	"excel":      {"PDF": -1, "XLSX": 51, "CSV": 62, "ODS": 60},
	"powerpoint": {"PDF": 32, "PPTX": 24, "PNG": 18, "ODP": 35},
}

// LibreOffice --convert-to names ("" = not offered without MS Office).
var loFormat = map[string]string{
	"PDF": "pdf", "DOCX": "docx", "TXT": "txt:Text (encoded):UTF8", "RTF": "rtf", "ODT": "odt",
	"XLSX": "xlsx", "CSV": "csv:Text - txt - csv (StarCalc):44,34,76", "ODS": "ods",
	"PPTX": "pptx", "ODP": "odp", "PNG": "",
}

func officeKind(path string) string { return officeExts[strings.ToLower(filepath.Ext(path))] }

// msOfficeHas is set by the Windows build: whether Word/Excel/PowerPoint
// is installed.
var msOfficeHas = func(app string) bool { return false }

func libreOffice() string {
	for _, v := range []string{"ProgramW6432", "ProgramFiles", "ProgramFiles(x86)"} {
		if d := os.Getenv(v); d != "" {
			p := filepath.Join(d, "LibreOffice", "program", "soffice.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	if p, err := exec.LookPath("soffice"); err == nil {
		return p
	}
	return ""
}

func officeItems(path string) []Item {
	kind := officeKind(path)
	own := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	ms, lo := msOfficeHas(kind), libreOffice() != ""
	var out []Item
	for _, t := range officeTargets[kind] {
		if t == own {
			continue
		}
		ok := ms || (lo && loFormat[t] != "")
		out = append(out, Item{ID: "to:" + t, Label: t, Enabled: ok})
	}
	return out
}

// officeScript drives the installed Office app invisibly. Macros are
// disabled, documents open read-only, and a placeholder password makes a
// protected document fail at once instead of waiting on a hidden password
// prompt. The app is closed only when it has nothing else open, so a Word,
// Excel or PowerPoint window the user is working in is never closed.
const officeScript = `$ErrorActionPreference = 'Stop'
$in = $env:FALAK_IN; $out = $env:FALAK_OUT; $fmt = [int]$env:FALAK_FMT
$m = [Reflection.Missing]::Value; $pw = 'falak-no-password'
switch ($env:FALAK_APP) {
 'word' {
  $a = New-Object -ComObject Word.Application; $a.Visible = $false; $a.DisplayAlerts = 0
  try { $a.AutomationSecurity = 3 } catch {}
  try { $d = $a.Documents.Open($in, $false, $true, $false, $pw, $m, $false, $m, $m, $m, $m, $false)
   $d.SaveAs2($out, $fmt); $d.Close(0) }
  finally { if ($a.Documents.Count -eq 0) { $a.Quit() } }
 }
 'excel' {
  $a = New-Object -ComObject Excel.Application; $a.Visible = $false; $a.DisplayAlerts = $false
  try { $a.AutomationSecurity = 3 } catch {}
  try { $b = $a.Workbooks.Open($in, 0, $true, $m, $pw, $m, $true)
   if ($fmt -lt 0) { $b.ExportAsFixedFormat(0, $out) } else { $b.SaveAs($out, $fmt) }
   $b.Close($false) }
  finally { if ($a.Workbooks.Count -eq 0) { $a.Quit() } }
 }
 'powerpoint' {
  $a = New-Object -ComObject PowerPoint.Application
  try { $a.AutomationSecurity = 3 } catch {}
  try { $p = $a.Presentations.Open($in, -1, 0, 0); $p.SaveAs($out, $fmt); $p.Close() }
  finally { if ($a.Presentations.Count -eq 0) { $a.Quit() } }
 }
}`

func runOffice(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	t := strings.TrimPrefix(id, "to:")
	kind := officeKind(path)
	if !strings.HasPrefix(id, "to:") || kind == "" {
		return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
	}
	abs, _ := filepath.Abs(path)
	// Office converts in one step and reports nothing: the bar follows the
	// expected time (starting the Office app is most of it).
	stop := estimate(progress, 0.02, 0.99, 5+fileMB(path)*0.8)
	defer stop()
	if t == "PNG" { // one image per slide, in a folder
		if !msOfficeHas(kind) {
			return "", errors.New(tr("Slides to images needs Microsoft PowerPoint", "تحويل الشرائح إلى صور يحتاج Microsoft PowerPoint"))
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		root := uniqueDir(filepath.Dir(path), base+" - PNG")
		tmp := tempFor(root)
		if err := officeMS(ctx, kind, abs, tmp, msFormat[kind][t]); err != nil {
			dropTemp(tmp)
			return "", err
		}
		return finishTemp(tmp, root)
	}
	dst := uniquePath(path, "", "."+strings.ToLower(t))
	tmp := tempFor(dst)
	var err error
	if msOfficeHas(kind) {
		err = officeMS(ctx, kind, abs, tmp, msFormat[kind][t])
	} else if lo := libreOffice(); lo != "" && loFormat[t] != "" {
		err = officeLO(ctx, lo, abs, tmp, loFormat[t])
	} else {
		err = errors.New(tr("This needs Microsoft Office or LibreOffice installed", "هذا يحتاج تثبيت Microsoft Office أو LibreOffice"))
	}
	if err != nil {
		dropTemp(tmp)
		return "", err
	}
	if _, err := os.Stat(tmp); err != nil {
		dropTemp(tmp)
		return "", errors.New(tr("The office program didn't produce a file", "لم يُنتج برنامج Office ملفًا"))
	}
	return finishTemp(tmp, dst)
}

func officeMS(ctx context.Context, app, in, out string, format int) error {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", officeScript)
	cmd.Env = append(os.Environ(), "FALAK_APP="+app, "FALAK_IN="+in, "FALAK_OUT="+out, fmt.Sprintf("FALAK_FMT=%d", format))
	PrepareCmd(cmd)
	if msg, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(tr("Microsoft Office couldn't convert the file: %s", "تعذّر على Microsoft Office تحويل الملف: %s"), lastLine(string(msg)))
	}
	return nil
}

func officeLO(ctx context.Context, soffice, in, out, filter string) error {
	dir := tempFor(out + ".lo")
	defer dropTemp(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, soffice, "--headless", "--norestore", "--convert-to", filter, "--outdir", dir, in)
	PrepareCmd(cmd)
	if msg, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(tr("LibreOffice couldn't convert the file: %s", "تعذّر على LibreOffice تحويل الملف: %s"), lastLine(string(msg)))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			return os.Rename(filepath.Join(dir, e.Name()), out)
		}
	}
	return errors.New(tr("LibreOffice didn't produce a file", "لم يُنتج LibreOffice ملفًا"))
}
