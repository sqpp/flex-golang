@echo off
SETLOCAL

IF "%1"=="build" GOTO Build
IF "%1"=="clean" GOTO Clean
IF "%1"=="test" GOTO Test
GOTO Help

:Build
echo Building flex-decode...
if not exist bin mkdir bin
go build -o bin/flex-decode.exe ./cmd/flex-decode
echo Build complete! Binary is in the bin\ directory.
GOTO End

:Clean
echo Cleaning...
if exist bin rmdir /s /q bin
echo Cleaned.
GOTO End

:Test
go test -v ./...
GOTO End

:Help
echo Usage: make.bat [build^|clean^|test]
echo   build - Compiles flex-decode.exe into bin\
echo   clean - Removes the bin\ directory
echo   test  - Runs tests

:End
ENDLOCAL
