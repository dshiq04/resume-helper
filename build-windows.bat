@echo off
setlocal
cd /d "%~dp0"

set "ARCH=%~1"
if "%ARCH%"=="" set "ARCH=amd64"
if /I not "%ARCH%"=="amd64" if /I not "%ARCH%"=="arm64" (
  echo [ERROR] Unsupported architecture: %ARCH%
  echo         Usage: build-windows.bat [amd64^|arm64]
  exit /b 1
)

echo ==^> Checking prerequisites...
where go >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Go was not found in PATH. Install Go 1.26+ first.
  exit /b 1
)
where npm >nul 2>nul
if errorlevel 1 (
  echo [ERROR] npm was not found in PATH. Install Node.js 20+ first.
  exit /b 1
)

if not exist "frontend\node_modules" (
  echo ==^> Installing frontend dependencies...
  call npm install --prefix frontend
  if errorlevel 1 exit /b 1
)

echo ==^> Building frontend...
call npm run build --prefix frontend
if errorlevel 1 exit /b 1

if not exist "bin" mkdir "bin"

set "OUT=bin\resume-helper.exe"
if /I not "%ARCH%"=="amd64" set "OUT=bin\resume-helper-%ARCH%.exe"

echo ==^> Building Windows (%ARCH%) binary...
set "GOOS=windows"
set "GOARCH=%ARCH%"
go build -trimpath -ldflags="-s -w -H windowsgui" -o "%OUT%" .
if errorlevel 1 exit /b 1

echo.
echo [OK] Built %OUT%
for %%F in ("%OUT%") do echo      Size: %%~zF bytes
endlocal
