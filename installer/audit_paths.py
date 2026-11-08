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

if fails:
    print("NSI PATH AUDIT FAILED:")
    for f in fails:
        print("  - " + f)
    sys.exit(1)
print("NSI PATH AUDIT PASSED: no forward-slash runtime paths; shortcuts,")
print("uninstaller deletes, WriteUninstaller order and Run value all correct.")
