#!/usr/bin/env python3
"""Static build check for installer/falak.nsi: fails (exit 1) if any
runtime Windows path written by the installer uses a forward slash
where a backslash is required (the bug that made the Start Menu
shortcut point to FalakFalak.exe), or if the shortcut/uninstaller
contract is violated. Run before every installer build:

    python3 installer/audit_paths.py
"""
import re
import sys
from pathlib import Path

fails = []
nsi = (Path(__file__).parent / "falak.nsi").read_text(encoding="utf-8")

# 1. No forward slash after any NSIS runtime path variable.
for m in re.finditer(r'\$(?:INSTDIR|SMPROGRAMS|DESKTOP|LOCALAPPDATA|TEMP|OUTDIR|STARTMENU|SMPROGRAMS)/', nsi):
    line_no = nsi.count("\n", 0, m.start()) + 1
    line = nsi.splitlines()[line_no - 1].strip()
    fails.append(f"line {line_no}: forward slash in runtime path: {line}")

# 1b. Compile-time asset paths use backslashes too: makensis on Windows
#     (3.13) reports File "${__FILEDIR__}/x" as "no files found".
for m in re.finditer(r'\$\{__FILEDIR__\}/', nsi):
    line_no = nsi.count("\n", 0, m.start()) + 1
    fails.append(f"line {line_no}: forward slash after ${{__FILEDIR__}}: {nsi.splitlines()[line_no - 1].strip()}")

# 2. Shortcuts must use the exact form: target, empty args, icon, index 0,
#    and must come AFTER Falak.exe is copied (File line).
want_smenu = '  CreateShortCut "$SMPROGRAMS\\Falak.lnk" "$INSTDIR\\Falak.exe" "" "$INSTDIR\\Falak.exe" 0'
want_desktop = '    CreateShortCut "$DESKTOP\\Falak.lnk" "$INSTDIR\\Falak.exe" "" "$INSTDIR\\Falak.exe" 0'
if want_smenu not in nsi:
    fails.append("missing exact Start Menu shortcut line:\n" + want_smenu)
if want_desktop not in nsi:
    fails.append("missing exact Desktop shortcut line:\n" + want_desktop)
file_line = '  File "${__FILEDIR__}\\Falak.exe"'
sc_line = nsi.find(want_smenu)
if file_line not in nsi:
    fails.append("missing exact line: " + file_line)
elif sc_line != -1 and nsi.find(file_line) > sc_line:
    fails.append("shortcuts must be created AFTER Falak.exe is copied")

# 3. Uninstaller must delete both shortcuts.
for lnk in ('Delete "$SMPROGRAMS\\Falak.lnk"', 'Delete "$DESKTOP\\Falak.lnk"'):
    if lnk not in nsi:
        fails.append("uninstaller does not delete: " + lnk)

# 4. WriteUninstaller (backslash form) must come before the registry writes.
wu = '  WriteUninstaller "$INSTDIR\\Uninstall.exe"'
if wu not in nsi:
    fails.append("missing: " + wu)
elif 'UninstallString' in nsi and nsi.find(wu) > nsi.find('"UninstallString"'):
    fails.append("WriteUninstaller must be called BEFORE the UninstallString writes")

# 5. The Run value must be a backslash path.
if "'\"$INSTDIR\\Falak.exe\" --startup'" not in nsi:
    fails.append("Run value must be '$INSTDIR\\Falak.exe' with a backslash")

# 6. The uninstaller must never delete $INSTDIR recursively (the user may
#    have picked a folder holding other files); only RMDir without /r.
if re.search(r'RMDir\s+/r\s+"\$INSTDIR', nsi):
    fails.append('uninstaller must not use RMDir /r on $INSTDIR')

# 7. No PowerShell in the installer (plain NSIS does the registry cleanup;
#    script hosts with -ExecutionPolicy Bypass look like malware to AV).
code = "\n".join(l for l in nsi.splitlines() if not l.lstrip().startswith((";", "#")))
if re.search(r'powershell', code, re.IGNORECASE):
    fails.append("installer must not run PowerShell")

# 8. The portable zip holds exactly Falak.exe, README.txt and
#    THIRD_PARTY_LICENSES.txt: the build script's list, and the built zip
#    when there is one.
import zipfile
PORTABLE = {"Falak.exe", "README.txt", "THIRD_PARTY_LICENSES.txt"}
here = Path(__file__).parent
mp = (here / "make_portable.py").read_text(encoding="utf-8")
listed = set(re.findall(r'^\s*"([^"]+)":\s*\w', mp.split("FILES = {", 1)[-1].split("}", 1)[0], re.MULTILINE))
if listed != PORTABLE:
    fails.append(f"make_portable.py FILES must be exactly {sorted(PORTABLE)}, got {sorted(listed)}")
if not (here / "portable" / "README.txt").is_file():
    fails.append("installer/portable/README.txt is missing")
built = here.parent / "build" / "Falak-1.0.0-portable.zip"
if built.is_file():
    with zipfile.ZipFile(built) as z:
        names = z.namelist()
    if sorted(names) != sorted(PORTABLE):
        fails.append(f"build/Falak-1.0.0-portable.zip must hold exactly {sorted(PORTABLE)}, got {names}")

if fails:
    print("NSI PATH AUDIT FAILED:")
    for f in fails:
        print("  - " + f)
    sys.exit(1)
print("NSI PATH AUDIT PASSED: no forward-slash runtime paths; shortcuts,")
print("uninstaller deletes, WriteUninstaller order and Run value all correct.")
