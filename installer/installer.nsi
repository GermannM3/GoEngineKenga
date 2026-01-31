; GoEngineKenga Installer — мастер установки в стиле современных программ (Welcome, каталог, опции, PATH)
; Сборка: makensis installer.nsi (из папки dist, где лежат exe и этот скрипт)

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"

; --------------------------------------------
!define APPNAME "GoEngineKenga"
!define COMPANYNAME "GoEngineKenga"
!define DESCRIPTION "Game engine and editor (Go)"
!define VERSIONMAJOR 1
!define VERSIONMINOR 0
!define VERSIONBUILD 0
!define HELPURL "https://github.com/GermannM3/GoEngineKenga"
!define UPDATEURL "https://github.com/GermannM3/GoEngineKenga/releases"
!define ABOUTURL "https://github.com/GermannM3/GoEngineKenga"
!define INSTALLSIZE 25000

RequestExecutionLevel admin
Unicode true
Name "${APPNAME}"
OutFile "GoEngineKenga-${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD}-setup.exe"
InstallDir "$PROGRAMFILES64\${APPNAME}"
InstallDirRegKey HKLM "Software\${APPNAME}" "InstallPath"

; --------------------------------------------
; Интерфейс мастера (как у современных установщиков)
!define MUI_ABORTWARNING
!define MUI_ICON ""
!define MUI_UNICON ""
!define MUI_HEADERIMAGE
!define MUI_WELCOMEPAGE_TITLE "Добро пожаловать в установку ${APPNAME}"
!define MUI_WELCOMEPAGE_TEXT "Этот мастер установит ${APPNAME} ${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD} — движок и редактор.$\r$\n$\r$\nРекомендуется закрыть все программы перед продолжением."
!define MUI_FINISHPAGE_TITLE "Установка завершена"
!define MUI_FINISHPAGE_TEXT "${APPNAME} установлен.$\r$\n$\r$\nНажмите Готово для выхода."
; Run checkbox only makes sense when editor is included (built with CGO)
!define MUI_FINISHPAGE_RUN "$INSTDIR\kenga-editor-windows-amd64.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Run ${APPNAME} Editor"
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_DIRECTORYPAGE_TEXT_TOP "Выберите папку для установки. Можно оставить по умолчанию."

; Страницы мастера
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "Russian"
!insertmacro MUI_LANGUAGE "English"

; --------------------------------------------
; Компоненты (опции установки)
!define SECTION_MAIN     "GoEngineKenga (движок и редактор)"
!define SECTION_DESKTOP  "Ярлык на рабочем столе"
!define SECTION_PATH     "Добавить в PATH (запуск kenga из командной строки)"
!define SECTION_START    "Пункты в меню Пуск"

Var AddToPath
Var AddDesktop
Var AddStartMenu

; --------------------------------------------
; Проверка прав администратора
!macro VerifyUserIsAdmin
UserInfo::GetAccountType
pop $0
${If} $0 != "admin"
    messageBox mb_iconstop "Требуются права администратора."
    setErrorLevel 740
    quit
${EndIf}
!macroend

Function .onInit
    setShellVarContext all
    !insertmacro VerifyUserIsAdmin
    StrCpy $AddToPath "1"
    StrCpy $AddDesktop "1"
    StrCpy $AddStartMenu "1"
FunctionEnd

; --------------------------------------------
; Секция "Основные файлы" (обязательная)
Section "!${SECTION_MAIN}" SEC_MAIN
    SectionIn RO
    SetOutPath $INSTDIR

    ; Editor optional (requires CGO to build)
    File /NONFATAL "kenga-editor-windows-amd64.exe"
    File "kenga-windows-amd64.exe"
    File "..\README.md"
    File "..\LICENSE"

    CreateDirectory "$INSTDIR\examples"
    CreateDirectory "$INSTDIR\examples\hello"
    CopyFiles "..\samples\hello\*" "$INSTDIR\examples\hello"

    ; Регистрация в «Установка и удаление программ»
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayName" "${APPNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "InstallLocation" "$INSTDIR"
    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" 0 +2
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayIcon" "$INSTDIR\kenga-editor-windows-amd64.exe"
    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" +2 0
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayIcon" "$INSTDIR\kenga-windows-amd64.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "Publisher" "${COMPANYNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "HelpLink" "${HELPURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLUpdateInfo" "${UPDATEURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLInfoAbout" "${ABOUTURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayVersion" "${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMajor" ${VERSIONMAJOR}
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMinor" ${VERSIONMINOR}
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoRepair" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "EstimatedSize" ${INSTALLSIZE}

    ; Путь установки для деинсталлятора
    WriteRegStr HKLM "Software\${APPNAME}" "InstallPath" "$INSTDIR"

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\kenga.exe" "" "$INSTDIR\kenga-windows-amd64.exe"
    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" 0 +2
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\kenga-editor.exe" "" "$INSTDIR\kenga-editor-windows-amd64.exe"

    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

; --------------------------------------------
; Секция «Ярлык на рабочем столе»
Section "${SECTION_DESKTOP}" SEC_DESKTOP
    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" 0 +2
    CreateShortCut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\kenga-editor-windows-amd64.exe" "" "$INSTDIR\kenga-editor-windows-amd64.exe" 0
SectionEnd

; --------------------------------------------
; Секция «Пункты в меню Пуск»
Section "${SECTION_START}" SEC_START
    CreateDirectory "$SMPROGRAMS\${APPNAME}"
    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" 0 +2
    CreateShortCut "$SMPROGRAMS\${APPNAME}\${APPNAME} Editor.lnk" "$INSTDIR\kenga-editor-windows-amd64.exe" "" "$INSTDIR\kenga-editor-windows-amd64.exe" 0
    CreateShortCut "$SMPROGRAMS\${APPNAME}\${APPNAME} CLI.lnk" "$INSTDIR\kenga-windows-amd64.exe" "" "$INSTDIR\kenga-windows-amd64.exe" 0
    CreateShortCut "$SMPROGRAMS\${APPNAME}\Удалить ${APPNAME}.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
SectionEnd

; --------------------------------------------
; Секция «Добавить в PATH» — через setx (лимит 1024 символа; при длинном PATH может не сработать)
Section "${SECTION_PATH}" SEC_PATH
    ReadEnvStr $0 PATH
    StrCpy $1 "$0;$INSTDIR"
    StrLen $3 $1
    IntCmp $3 1024 0 0 skip_setx
    DetailPrint "PATH: слишком длинный для setx, добавьте вручную: $INSTDIR"
    Goto path_done
    skip_setx:
    ExecWait 'cmd /c setx PATH "$1"' $2
    ${If} $2 == 0
        DetailPrint "PATH: добавлен $INSTDIR (перезапустите терминал)"
    ${Else}
        DetailPrint "PATH: не изменён (код $2). Добавьте вручную: $INSTDIR"
    ${EndIf}
    path_done:
SectionEnd

; --------------------------------------------
; Условная установка по умолчанию: все включены
Function .onSelChange
    ; Оставляем основную секцию всегда выбранной (SectionIn RO)
FunctionEnd

; --------------------------------------------
; Деинсталлятор
Section "Uninstall"
    ; PATH при установке через setx — при удалении не трогаем PATH (пользователь может почистить вручную при необходимости)

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\kenga.exe"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\App Paths\kenga-editor.exe"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
    DeleteRegKey HKLM "Software\${APPNAME}"

    IfFileExists "$INSTDIR\kenga-editor-windows-amd64.exe" 0 +2
    Delete "$INSTDIR\kenga-editor-windows-amd64.exe"
    Delete "$INSTDIR\kenga-windows-amd64.exe"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\uninstall.exe"
    RMDir /r "$INSTDIR\examples"
    RMDir "$INSTDIR"

    Delete "$DESKTOP\${APPNAME}.lnk"
    RMDir /r "$SMPROGRAMS\${APPNAME}"
SectionEnd
