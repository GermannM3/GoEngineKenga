@echo off
REM Build Windows installer. Run from repo root. Requires: dist\*.exe and NSIS.
cd /d "%~dp0.."
powershell -ExecutionPolicy Bypass -File "%~dp0build-installer.ps1" -Version "1.0.0"
pause
