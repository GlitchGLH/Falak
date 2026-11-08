; Falak 1.0.0 installer - per-user, no admin rights needed
Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

!define RunKey "Software\Microsoft\Windows\CurrentVersion\Run"
!define UninstKey "Software\Microsoft\Windows\CurrentVersion\Uninstall\Falak"
!define LayersKey "Software\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers"

Name "Falak"
OutFile "${__FILEDIR__}\..\..\FalakSetup-1.0.0.exe"
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

!define MUI_ICON "${__FILEDIR__}\raha-nsis.ico"
!define MUI_UNICON "${__FILEDIR__}\raha-nsis.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\Falak.exe"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${__FILEDIR__}\license.txt"
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

; wmQuitUpdate in main_windows.go (WM_APP + 9): asks a running Falak to
; close cleanly (it stops its conversions first), as a newer copy does.
!define FALAK_QUIT_MSG 0x8009

; Close a running Falak: send FalakMsg the quit message and wait up to 5 s.
; Only if Falak.exe is still running then, end it with taskkill /F, and give
; any copy that cannot be ended (the key helper runs as administrator; it
; exits by itself once Falak is gone) up to 3 s more.
; Every nsExec::Exec pushes its exit code; each one is popped so nothing is
; left on the stack for later code to read by mistake.
!macro FalakRunning
  nsExec::Exec 'cmd /c tasklist /FI "IMAGENAME eq Falak.exe" | find /I "Falak.exe" > NUL'
  Pop $0 ; 0 = a Falak.exe is running
!macroend
!macro CloseFalak
  FindWindow $1 "FalakMsg"
  ${If} $1 <> 0
    SendMessage $1 ${FALAK_QUIT_MSG} 0 0 /TIMEOUT=2000
  ${EndIf}
  StrCpy $2 0
  ${Do}
    !insertmacro FalakRunning
    ${IfThen} $0 != 0 ${|} ${Break} ${|}
    ${If} $2 = 25 ; 25 x 200 ms = 5 s
      nsExec::Exec 'taskkill /F /IM Falak.exe'
      Pop $0
    ${EndIf}
    ${IfThen} $2 >= 40 ${|} ${Break} ${|}
    Sleep 200
    IntOp $2 $2 + 1
  ${Loop}
!macroend

; Earlier builds created the scheduled task Falak\KeysHelper, which started
; Falak.exe as administrator without asking. Remove it on install and on
; uninstall. If Windows refuses (the task was created as administrator), it
; is deleted through one administrator prompt; nothing is asked otherwise.
!macro DeleteOldHelperTask
  nsExec::Exec 'schtasks /Query /TN "Falak\KeysHelper"'
  Pop $0
  ${If} $0 == 0
    nsExec::Exec 'schtasks /Delete /TN "Falak\KeysHelper" /F'
    Pop $0
    ${If} $0 != 0
      ExecShellWait "runas" "$SYSDIR\schtasks.exe" '/Delete /TN "Falak\KeysHelper" /F' SW_HIDE
    ${EndIf}
  ${EndIf}
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
  !insertmacro DeleteOldHelperTask
  SetOutPath "$INSTDIR"
  File "${__FILEDIR__}\Falak.exe"
  File "${__FILEDIR__}\..\THIRD_PARTY_LICENSES.txt"
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
  ; MessageBox reports the answer by jumping, not on the stack. Yes (and a
  ; silent uninstall, via /SD) keeps the settings; No deletes them.
  MessageBox MB_YESNO "$(DelSettings)" /SD IDYES IDYES keepSettings
  DeleteRegKey HKCU "Software\Falak"
  keepSettings:
  DeleteRegValue HKCU "${RunKey}" "Falak"
  DeleteRegValue HKCU "${RunKey}" "Raha"
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run" "Falak"
  ; Remove "Run as administrator" marks on any ...\Falak.exe (plain NSIS,
  ; no PowerShell). StrCmp/== ignore case. After a delete the same index
  ; holds the next value, so the index only moves on when a value is kept.
  StrCpy $1 0
  ${Do}
    ClearErrors
    EnumRegValue $2 HKCU "${LayersKey}" $1
    ${If} ${Errors}
    ${OrIf} $2 == ""
      ${Break}
    ${EndIf}
    StrCpy $3 $2 "" -10
    ${If} $3 == "\Falak.exe"
      DeleteRegValue HKCU "${LayersKey}" $2
    ${Else}
      IntOp $1 $1 + 1
    ${EndIf}
  ${Loop}
  !insertmacro DeleteOldHelperTask
  Delete "$SMPROGRAMS\Falak.lnk"
  Delete "$DESKTOP\Falak.lnk"
  RMDir "$SMPROGRAMS"
  ; Only the files this installer put in $INSTDIR (plus Falak's own
  ; self-update leftovers), never the whole folder: the user may have picked
  ; a folder that holds other files. The folder goes only if it is now empty.
  Delete "$INSTDIR\Falak.exe"
  Delete "$INSTDIR\Falak.exe.new"
  Delete "$INSTDIR\Falak.exe.old"
  Delete "$INSTDIR\THIRD_PARTY_LICENSES.txt"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  RMDir /r "$LOCALAPPDATA\Falak"
  RMDir /r "$TEMP\falak-tools"
  DeleteRegKey HKCU "${UninstKey}"
SectionEnd
