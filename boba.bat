@echo off
go build -o boba-text.exe main.go || exit /b 1
boba-text.exe %*
