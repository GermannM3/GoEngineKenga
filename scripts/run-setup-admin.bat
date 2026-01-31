@echo off
REM Run setup-dev-env.ps1 as Administrator (opens new elevated PowerShell)
powershell -Command "Start-Process powershell -ArgumentList '-ExecutionPolicy Bypass -File \"%~dp0setup-dev-env.ps1\"' -Verb RunAs"
