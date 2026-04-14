; MCP-Tester — Windows Installer
; Installs the mcp-tester CLI and adds it to the system PATH.
;
; Requires: NSIS + MUI2, WordFunc (makensis)
; Build:    makensis scripts/windows-installer.nsi  (run from repo root)

; Set working directory to project root (script lives in scripts/)
!cd ".."

!define APP_NAME    "MCP-Tester"
!define APP_VERSION "0.2.3"
!define PUBLISHER   "Michael Lechner"
!define COPYRIGHT   "Copyright (c) 2026 Michael Lechner"
!define INSTALL_DIR "$PROGRAMFILES64\mcp-tester"
!define REG_KEY     "Software\Microsoft\Windows\CurrentVersion\Uninstall\mcp-tester"

Name "${APP_NAME} ${APP_VERSION}"
OutFile "dist\mcp-tester-setup-v${APP_VERSION}-windows-amd64.exe"
InstallDir "${INSTALL_DIR}"
InstallDirRegKey HKLM "${REG_KEY}" "InstallLocation"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
Unicode true

; ── Pages ─────────────────────────────────────────────────────────────────────
!include "MUI2.nsh"
!include "WordFunc.nsh"

!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "German"

; ── Install ───────────────────────────────────────────────────────────────────
Section "MCP-Tester CLI" SecCLI
  SectionIn RO  ; required

  SetOutPath "$INSTDIR"
  File "bin\mcp-tester-windows-amd64.exe"
  ; Rename to plain mcp-tester.exe so it's callable without the arch suffix
  Rename "$INSTDIR\mcp-tester-windows-amd64.exe" "$INSTDIR\mcp-tester.exe"
  File "README.md"
  File "LICENSE"

  ; Add to system PATH (skip if already present)
  ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ${WordFind} "$0" "$INSTDIR" "E+1{" $1
  IfErrors 0 already_in_path
    WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0;$INSTDIR"
    SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
  already_in_path:

  ; Registry for Add/Remove Programs
  WriteRegStr   HKLM "${REG_KEY}" "DisplayName"     "${APP_NAME} ${APP_VERSION}"
  WriteRegStr   HKLM "${REG_KEY}" "DisplayVersion"  "${APP_VERSION}"
  WriteRegStr   HKLM "${REG_KEY}" "Publisher"       "${PUBLISHER}"
  WriteRegStr   HKLM "${REG_KEY}" "Comments"        "${COPYRIGHT}"
  WriteRegStr   HKLM "${REG_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr   HKLM "${REG_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegDWORD HKLM "${REG_KEY}" "NoModify"        1
  WriteRegDWORD HKLM "${REG_KEY}" "NoRepair"        1

  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; ── Uninstall ─────────────────────────────────────────────────────────────────
Section "Uninstall"
  Delete "$INSTDIR\mcp-tester.exe"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\uninstall.exe"
  RMDir  "$INSTDIR"

  ; Remove from PATH
  ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ${WordReplace} "$0" ";$INSTDIR" "" "+" $0
  ${WordReplace} "$0" "$INSTDIR;" "" "+" $0
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0"
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  DeleteRegKey HKLM "${REG_KEY}"
SectionEnd
