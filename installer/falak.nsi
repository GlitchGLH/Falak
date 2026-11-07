; Falak 1.0.0 installer - per-user, no admin rights needed
Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define RunKey "Software\Microsoft\Windows\CurrentVersion\Run"
!define UninstKey "Software\Microsoft\Windows\CurrentVersion\Uninstall\Falak"

Name "Falak"
OutFile "${__FILEDIR__}/../../FalakSetup-1.0.0.exe"
InstallDir "$LOCALAPPDATA\Programs\Falak"
RequestExecutionLevel user
ShowInstDetails show
ShowUnInstDetails show
SetCompressor /SOLID lzma

BrandingText "Falak 1.0.0 · By UncleGlitch"

VIProductVersion "1.0.0.0"
VIAddVersionKey /LANG=1033 FileVersion "1.0.0.0"
VIAddVersionKey /LANG=1033 ProductVersion "1.0.0.0"
VIAddVersionKey /LANG=1033 ProductName "Falak"
VIAddVersionKey /LANG=1033 CompanyName "UncleGlitch"
VIAddVersionKey /LANG=1033 FileDescription "Falak installer"
VIAddVersionKey /LANG=1033 LegalCopyright "(c) 2026 UncleGlitch"
VIAddVersionKey /LANG=1033 OriginalFilename "FalakSetup-1.0.0.exe"

!define MUI_ICON "${__FILEDIR__}/raha-nsis.ico"
!define MUI_UNICON "${__FILEDIR__}/raha-nsis.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\Falak.exe"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${__FILEDIR__}/license.txt"
!insertmacro MUI_PAGE_DIRECTORY
Page custom OptionsPage OptionsPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "Arabic"

LangString OptTitle ${LANG_ENGLISH} "Options"
LangString OptText ${LANG_ENGLISH} "Choose the shortcuts you want."
LangString OptDesktop ${LANG_ENGLISH} "Create a desktop shortcut"
LangString OptStartup ${LANG_ENGLISH} "Start Falak when you sign in to Windows"
LangString DelSettings ${LANG_ENGLISH} "Keep your Falak settings?$\n(Choose No to delete them.)"
LangString OptTitle ${LANG_ARABIC} "الخيارات"
LangString OptText ${LANG_ARABIC} "اختر الاختصارات التي تريدها."
LangString OptDesktop ${LANG_ARABIC} "إنشاء اختصار على سطح المكتب"
LangString OptStartup ${LANG_ARABIC} "تشغيل فلك عند تسجيل الدخول إلى ويندوز"
LangString DelSettings ${LANG_ARABIC} "هل تريد الاحتفاظ بإعدادات فلك؟$\n(اختر «لا» لحذفها.)"

Var DesktopCk
Var StartupCk
Var hOptDlg
Var DesktopHnd
Var StartupHnd

; Close a running Falak quietly: graceful close, wait 3 s, then end it.
!macro CloseFalak
  nsExec::Exec 'schtasks /End /TN "Falak\KeysHelper"'
  nsExec::Exec 'taskkill /IM Falak.exe'
  Sleep 3000
  nsExec::Exec 'cmd /c tasklist /FI "IMAGENAME eq Falak.exe" | find /I "Falak.exe" > NUL'
  Pop $0
  ${If} $0 == 0
    nsExec::Exec 'taskkill /F /IM Falak.exe'
  ${EndIf}
  Sleep 500
!macroend

Function .onInit
  SetShellVarContext current
  StrCpy $DesktopCk 0
  StrCpy $StartupCk 1
FunctionEnd

Function un.onInit
  SetShellVarContext current
FunctionEnd

Function OptionsPage
  !insertmacro MUI_HEADER_TEXT "$(OptTitle)" "$(OptText)"
  nsDialogs::Create 1018
  Pop $hOptDlg
  ${NSD_CreateCheckbox} 0 12u 100% 12u "$(OptDesktop)"
  Pop $DesktopHnd
  ${NSD_SetState} $DesktopHnd $DesktopCk
  ${NSD_CreateCheckbox} 0 32u 100% 12u "$(OptStartup)"
  Pop $StartupHnd
  ${NSD_SetState} $StartupHnd $StartupCk
  nsDialogs::Show
FunctionEnd

Function OptionsPageLeave
  ${NSD_GetState} $DesktopHnd $DesktopCk
  ${NSD_GetState} $StartupHnd $StartupCk
FunctionEnd

Section Install
  !insertmacro CloseFalak
  SetOutPath "$INSTDIR"
  File "${__FILEDIR__}/Falak.exe"
  File "${__FILEDIR__}/../THIRD_PARTY_LICENSES.txt"
  CreateShortCut "$SMPROGRAMS\Falak.lnk" "$INSTDIR\Falak.exe" "" "$INSTDIR\Falak.exe" 0
  ${If} $DesktopCk == 1
    CreateShortCut "$DESKTOP\Falak.lnk" "$INSTDIR\Falak.exe" "" "$INSTDIR\Falak.exe" 0
  ${EndIf}
  ${If} $StartupCk == 1
    WriteRegStr HKCU "${RunKey}" "Falak" '"$INSTDIR\Falak.exe" --startup'
    WriteRegBin HKCU "Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run" "Falak" 020000000000000000000000
  ${Else}
    DeleteRegValue HKCU "${RunKey}" "Falak"
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run" "Falak"
  ${EndIf}
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "${UninstKey}" "DisplayName" "Falak"
  WriteRegStr HKCU "${UninstKey}" "DisplayVersion" "1.0.0"
  WriteRegStr HKCU "${UninstKey}" "Publisher" "UncleGlitch"
  WriteRegStr HKCU "${UninstKey}" "DisplayIcon" "$INSTDIR\Falak.exe"
  WriteRegStr HKCU "${UninstKey}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UninstKey}" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
  WriteRegStr HKCU "${UninstKey}" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"
  WriteRegStr HKCU "${UninstKey}" "URLInfoAbout" "https://github.com/GlitchGLH/Falak"
  WriteRegDWORD HKCU "${UninstKey}" "NoModify" 1
  WriteRegDWORD HKCU "${UninstKey}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/M=Falak.exe /S=0K /B=0" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "${UninstKey}" "EstimatedSize" $0
SectionEnd

Section Uninstall
  !insertmacro CloseFalak
  ${IfNot} ${Silent}
    MessageBox MB_YESNO "$(DelSettings)"
    Pop $0
  ${EndIf}
  ${If} $0 == 7
    DeleteRegKey HKCU "Software\Falak"
  ${EndIf}
  DeleteRegValue HKCU "${RunKey}" "Falak"
  DeleteRegValue HKCU "${RunKey}" "Raha"
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run" "Falak"
  FileOpen $1 "$TEMP\falak-cleanup.ps1" w
  FileWrite $1 "$$p='HKCU:\\Software\\Microsoft\\Windows NT\\CurrentVersion\\AppCompatFlags\\Layers'$\r$\n"
  FileWrite $1 "$$k=Get-Item -LiteralPath $$p -ErrorAction SilentlyContinue$\r$\n"
  FileWrite $1 "if($$k){$$k.Property | Where-Object {$$_ -like '*\\Falak.exe'} | ForEach-Object {Remove-ItemProperty -LiteralPath $$p -Name $$_}}$\r$\n"
  FileClose $1
  nsExec::Exec 'powershell -NoProfile -ExecutionPolicy Bypass -File "$TEMP\falak-cleanup.ps1"'
  Pop $0
  Delete "$TEMP\falak-cleanup.ps1"
  Pop $0
  nsExec::Exec 'schtasks /Delete /TN "Falak\KeysHelper" /F'
  Delete "$SMPROGRAMS\Falak.lnk"
  Delete "$DESKTOP\Falak.lnk"
  RMDir "$SMPROGRAMS"
  RMDir /r "$INSTDIR"
  RMDir /r "$LOCALAPPDATA\Falak"
  RMDir /r "$TEMP\falak-tools"
  DeleteRegKey HKCU "${UninstKey}"
SectionEnd
