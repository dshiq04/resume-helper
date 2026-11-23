@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo.
echo ============================================
echo  ResumeTrail password recovery
echo ============================================
echo.
go run ./cmd/recover-password
echo.
pause
