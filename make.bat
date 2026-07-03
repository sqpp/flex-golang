@echo off
setlocal EnableDelayedExpansion

REM Default version if not provided
if "%VERSION%"=="" set VERSION=1.0.0

REM Build metadata
for /f %%i in ('powershell -NoProfile -Command "Get-Date -Format yyyy-MM-dd_HH:mm:ss"') do set BUILD_TIME=%%i
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=unknown

for /f %%i in ('go env GOOS') do set GOOS_VAL=%%i
for /f %%i in ('go env GOARCH') do set GOARCH_VAL=%%i
set BUILD_ARCH=%GOOS_VAL%/%GOARCH_VAL%

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i

REM LDFLAGS
set LDFLAGS=-X github.com/sqpp/flex-golang.Version=%VERSION% ^
 -X github.com/sqpp/flex-golang.BuildTime=%BUILD_TIME% ^
 -X github.com/sqpp/flex-golang.GitCommit=%GIT_COMMIT% ^
 -X github.com/sqpp/flex-golang.Author=marcell ^
 -X github.com/sqpp/flex-golang.ProjectURL=https://pagercast.com ^
 -X github.com/sqpp/flex-golang.BuildArch=%BUILD_ARCH% ^
 -X github.com/sqpp/flex-golang.BuildGoVer=%GO_VERSION%

REM Create bin folder
if not exist bin mkdir bin

if "%1"=="" goto build

if "%1"=="build" goto build
if "%1"=="install" goto install
if "%1"=="test" goto test
if "%1"=="clean" goto clean
if "%1"=="version" goto version
if "%1"=="cross-compile" goto cross
if "%1"=="help" goto help

:build
echo Building FLEX-GO v%VERSION%...
go build -ldflags "%LDFLAGS%" -o bin\flex-encode.exe ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode.exe ./cmd/flex-decode
echo Build complete!
goto end

:install
go install -ldflags "%LDFLAGS%" ./cmd/flex-encode
go install -ldflags "%LDFLAGS%" ./cmd/flex-decode
goto end

:test
go test -v ./...
goto end

:clean
rmdir /s /q bin
goto end

:version
echo Version: %VERSION%
echo Build Time: %BUILD_TIME%
echo Git Commit: %GIT_COMMIT%
goto end

:cross
echo Cross-compiling...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o bin\flex-encode-linux-amd64 ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode-linux-amd64 ./cmd/flex-decode

set GOOS=linux
set GOARCH=arm64
go build -ldflags "%LDFLAGS%" -o bin\flex-encode-linux-arm64 ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode-linux-arm64 ./cmd/flex-decode

set GOOS=windows
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o bin\flex-encode-windows-amd64.exe ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode-windows-amd64.exe ./cmd/flex-decode

set GOOS=darwin
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o bin\flex-encode-darwin-amd64 ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode-darwin-amd64 ./cmd/flex-decode

set GOOS=darwin
set GOARCH=arm64
go build -ldflags "%LDFLAGS%" -o bin\flex-encode-darwin-arm64 ./cmd/flex-encode
go build -ldflags "%LDFLAGS%" -o bin\flex-decode-darwin-arm64 ./cmd/flex-decode

echo Cross-compilation complete!
goto end

:help
echo Available targets:
echo build
echo install
echo test
echo clean
echo version
echo cross-compile
echo help
goto end

:end
endlocal
