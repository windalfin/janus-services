@echo off
setlocal

rem Load environment variables from .env
for /f "tokens=*" %%a in (.env) do set %%a

rem Run the program
go run cmd\processor\main.go

endlocal