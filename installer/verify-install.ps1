# Falak install verification (run on real Windows, NOT in the build sandbox):
#   powershell -ExecutionPolicy Bypass -File verify-install.ps1 [-Setup <path to FalakSetup-1.0.0.exe>]
# It silently installs Falak (/S, never waits for a GUI), prints every
# shortcut target and registry value, verifies each referenced file exists,
# then silently uninstalls and verifies everything is gone.
# Exits 0 = all checks passed, 1 = at least one check failed.
param([string]$Setup = "")

$ErrorActionPreference = 'Continue'
$fail = 0
function Check($ok, $msg) {
    if ($ok) { Write-Host "  [PASS] $msg" } else { Write-Host "  [FAIL] $msg"; $script:fail = 1 }
}
if (-not $Setup) {
    $cand = Get-ChildItem -Path "$PSScriptRoot\..\..", "$PSScriptRoot" -Filter 'FalakSetup-1.0.0.exe' -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($cand) { $Setup = $cand.FullName } else { Write-Host "[FAIL] FalakSetup-1.0.0.exe not found; pass -Setup <path>"; exit 1 }
}
Write-Host "== Silent install from $Setup =="
Start-Process -Wait -FilePath $Setup -ArgumentList '/S'
$inst = "$env:LOCALAPPDATA\Programs\Falak"
Check (Test-Path "$inst\Falak.exe") "installed $inst\Falak.exe"
Check (Test-Path "$inst\Uninstall.exe") "installed $inst\Uninstall.exe"
Check (Test-Path "$inst\THIRD_PARTY_LICENSES.txt") "installed THIRD_PARTY_LICENSES.txt"

Write-Host "== Shortcuts (targets must exist and end with \Falak.exe) =="
$sh = New-Object -ComObject WScript.Shell
foreach ($lnk in @("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Falak.lnk", "$env:USERPROFILE\Desktop\Falak.lnk")) {
    if (Test-Path $lnk) {
        $l = $sh.CreateShortcut($lnk)
        Write-Host "  $lnk"
        Write-Host "    Target : $($l.TargetPath)"
        Write-Host "    Icon   : $($l.IconLocation)"
        Check ($l.TargetPath -ieq "$inst\Falak.exe") "target is exactly $inst\Falak.exe"
        Check (Test-Path -LiteralPath $l.TargetPath) "target file exists"
        Check ($l.TargetPath -imatch '\\Falak\.exe$') "no eaten separator (would be ...FalakFalak.exe)"
    } else {
        Write-Host "  [INFO] $lnk not created (desktop shortcut is optional; Start Menu one is expected)"
        if ($lnk -like '*Start Menu*') { Check $false "Start Menu shortcut missing" }
    }
}

Write-Host "== Registry: Uninstall key =="
$k = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\Falak'
$v = Get-ItemProperty -Path $k -ErrorAction SilentlyContinue
if ($v) {
    $v.PSObject.Properties | Where-Object { $_.Name -notmatch '^PS' } | ForEach-Object { Write-Host "  $($_.Name) = $($_.Value)" }
    Check ($v.UninstallString -match '^"[^"]+\\Uninstall\.exe"$') "UninstallString is quoted with backslashes"
    $ustarget = $v.UninstallString.Trim('"')
    Check (Test-Path -LiteralPath $ustarget) "UninstallString target exists"
    Check ($v.DisplayIcon -ieq "$inst\Falak.exe") "DisplayIcon correct"
    Check ($v.InstallLocation -ieq $inst) "InstallLocation correct"
    Check (-not ($v.PSObject.Properties.Name -contains 'NoRemove')) "NoRemove NOT present"
} else { Check $false "Uninstall key missing" }

Write-Host "== Registry: Run value + display affinity =="
$run = (Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -ErrorAction SilentlyContinue).Falak
if ($run) {
    Write-Host "  Run\Falak = $run"
    $m = [regex]::Match($run, '^"([^"]+)"')
    if ($m.Success -and (Test-Path -LiteralPath $m.Groups[1].Value)) { Check $true "Run target file exists" } else { Check $false "Run target file exists" }
} else { Check $false "Run value missing" }
$aff = Get-ItemProperty 'HKCU:\SOFTWARE\Microsoft\...\0' -ErrorAction SilentlyContinue 2>$null
Write-Host "  (window affinity is logged by Falak itself: falak.log lines 'display affinity 0 (WDA_NONE)')"

Write-Host "== Silent uninstall =="
Start-Process -Wait -FilePath "$inst\Uninstall.exe" -ArgumentList '/S'
Start-Sleep -Seconds 2
Check (-not (Test-Path $inst)) "install folder removed"
Check (-not (Test-Path "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Falak.lnk")) "Start Menu shortcut removed"
Check (-not (Test-Path "$env:USERPROFILE\Desktop\Falak.lnk")) "Desktop shortcut removed"
Check (-not (Test-Path $k)) "Uninstall key removed"
$runAfter = (Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -ErrorAction SilentlyContinue).Falak
Check (-not $runAfter) "Run value removed"

if ($fail) { Write-Host "== RESULT: FAILED =="; exit 1 }
Write-Host "== RESULT: ALL CHECKS PASSED =="
exit 0
