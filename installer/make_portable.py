#!/usr/bin/env python3
"""Builds build/Falak-1.0.0-portable.zip from exactly three files:

    Falak.exe                 (build/Falak.exe, built beforehand)
    README.txt                (installer/portable/README.txt)
    THIRD_PARTY_LICENSES.txt  (repo root)

Run from anywhere, after building Falak.exe into build/:

    py installer/make_portable.py

The files sit at the top of the zip with a fixed timestamp, so the same
inputs always give the same zip. audit_paths.py checks the file list.
"""
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / "build"
ZIP = BUILD / "Falak-1.0.0-portable.zip"

# name in the zip -> source file (audit_paths.py reads this list)
FILES = {
    "Falak.exe": BUILD / "Falak.exe",
    "README.txt": ROOT / "installer" / "portable" / "README.txt",
    "THIRD_PARTY_LICENSES.txt": ROOT / "THIRD_PARTY_LICENSES.txt",
}

STAMP = (2026, 11, 8, 0, 0, 0)


def main():
    missing = [str(p) for p in FILES.values() if not p.is_file()]
    if missing:
        sys.exit("missing: " + ", ".join(missing))
    tmp = ZIP.with_suffix(".zip.part")
    with zipfile.ZipFile(tmp, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for name, src in FILES.items():
            info = zipfile.ZipInfo(name, STAMP)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            z.writestr(info, src.read_bytes(), compresslevel=9)
    tmp.replace(ZIP)
    with zipfile.ZipFile(ZIP) as z:
        names = z.namelist()
    if sorted(names) != sorted(FILES):
        sys.exit(f"unexpected zip contents: {names}")
    print(f"{ZIP} ({ZIP.stat().st_size} bytes): {', '.join(names)}")


if __name__ == "__main__":
    main()
