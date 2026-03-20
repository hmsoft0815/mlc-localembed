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
    File "../../bin-windows\localembed-gui.exe"
    File "../../bin-windows\cli.exe"
    File "../../bin-windows\preloader.exe"
    
    ; Library
    File "../../onnxruntime.dll"
    
    ; Docs & Config
    File "../../LICENSE"
    File "../../README.md"
    File "../../config.yaml"

    ; Uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Start Menu Shortcuts
    CreateDirectory "$SMPROGRAMS\LocalEmbed"
    CreateShortcut "$SMPROGRAMS\LocalEmbed\LocalEmbed Manager.lnk" "$INSTDIR\localembed-gui.exe"
    CreateShortcut "$SMPROGRAMS\LocalEmbed\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
    
    ; Desktop Shortcut (Optional)
    CreateShortcut "$DESKTOP\LocalEmbed Manager.lnk" "$INSTDIR\localembed-gui.exe"

    ; Registry keys for Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "DisplayName" "LocalEmbed"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "QuietUninstallString" "$\"$INSTDIR\Uninstall.exe$\" /S"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "DisplayIcon" "$INSTDIR\localembed-gui.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed" "Publisher" "Michael Lechner"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\mlcembedder.exe"
    Delete "$INSTDIR\localembed-gui.exe"
    Delete "$INSTDIR\cli.exe"
    Delete "$INSTDIR\preloader.exe"
    Delete "$INSTDIR\onnxruntime.dll"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\config.yaml"
    Delete "$INSTDIR\Uninstall.exe"

    RMDir "$INSTDIR"

    Delete "$SMPROGRAMS\LocalEmbed\LocalEmbed Manager.lnk"
    Delete "$SMPROGRAMS\LocalEmbed\Uninstall.lnk"
    RMDir "$SMPROGRAMS\LocalEmbed"
    
    Delete "$DESKTOP\LocalEmbed Manager.lnk"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\LocalEmbed"
SectionEnd
