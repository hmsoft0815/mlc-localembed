!include "MUI2.nsh"

Name "LocalEmbed"
OutFile "../../release_pkgs/localembed-setup.exe"
InstallDir "$PROGRAMFILES64\LocalEmbed"
RequestExecutionLevel admin

!define MUI_ABORTWARNING

; --- Pages ---
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "German"

Section "Install"
    SetOutPath "$INSTDIR"
    
    ; Binaries (from bin-windows directory)
    File "../../bin-windows\mlcembedder.exe"
    File "../../bin-windows\cli.exe"
    File "../../bin-windows\preloader.exe"
    File "../../bin-windows\onnxruntime.dll"
    File "../../bin-windows\LICENSE"
    File "../../bin-windows\README.md"
    File "../../bin-windows\config.yaml"

    ; Starter batch file (foreground launcher; for autostart use Task Scheduler or nssm)
    File "../../bin-windows\start.bat"

    ; Uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Start Menu Shortcuts
    CreateDirectory "$SMPROGRAMS\LocalEmbed"
    CreateShortcut "$SMPROGRAMS\LocalEmbed\LocalEmbed (start server).lnk" "$INSTDIR\start.bat"
    CreateShortcut "$SMPROGRAMS\LocalEmbed\Uninstall.lnk" "$INSTDIR\Uninstall.exe"

    ; Registry keys for Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "DisplayName" "LocalEmbed"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "DisplayIcon" "$INSTDIR\mlcembedder.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "Publisher" "Michael Lechner"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\mlcembedder.exe"
    Delete "$INSTDIR\cli.exe"
    Delete "$INSTDIR\preloader.exe"
    Delete "$INSTDIR\onnxruntime.dll"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\config.yaml"
    Delete "$INSTDIR\start.bat"
    Delete "$INSTDIR\Uninstall.exe"

    RMDir "$INSTDIR"

    Delete "$SMPROGRAMS\LocalEmbed\LocalEmbed (start server).lnk"
    Delete "$SMPROGRAMS\LocalEmbed\Uninstall.lnk"
    RMDir "$SMPROGRAMS\LocalEmbed"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed"
SectionEnd
