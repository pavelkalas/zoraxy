@echo off
chcp 65001 >nul
title Zoraxy - nacist image

REM ============================================================
REM  load.bat
REM
REM  Spust na CILOVEM stroji (kde chces zoraxy provozovat).
REM  Nacte image z build/output/zoraxy-multi.tar do dockeru.
REM  Predtim tam musis mit nakopirovanou slozku build/output.
REM ============================================================

set "IMAGE_NAME=zoraxy-multi"

echo.
echo  ================================================
echo   Zoraxy - nacitam image z taru
echo  ================================================
echo.

if not exist "%~dp0output\%IMAGE_NAME%.tar" (
    echo   CHYBA: Soubor output\%IMAGE_NAME%.tar neexistuje.
    echo   Zkopiruj sem slozku output/ z build stroje.
    pause
    exit /b 1
)

docker load -i "%~dp0output\%IMAGE_NAME%.tar"
if errorlevel 1 (
    echo   CHYBA: docker load selhal.
    pause
    exit /b 1
)

echo.
echo  ================================================
echo   Hotovo! Image je nacteny jako %IMAGE_NAME%:latest.
echo.
echo   Ted v docker-compose.yml zmen image na: %IMAGE_NAME%:latest
echo   a spus:  docker compose up -d
echo  ================================================
echo.
pause
