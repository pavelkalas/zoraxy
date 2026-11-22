@echo off
setlocal
chcp 65001 >nul
title Zoraxy - push do registru

REM ============================================================
REM  push.bat  (VOLITELNE)
REM
REM  Nahraje image do Docker registru, abys ho pak mohl na serveru
REM  jen "docker pull" (bez kopirovani taru).
REM
REM  Dole si nastav REGISTRY:
REM    - Docker Hub:  tvujlogin/zoraxy-multi
REM    - GHCR:        ghcr.io/tvujlogin/zoraxy-multi
REM    - vlastni:     registry.tvoje-domena.cz:5000/zoraxy-multi
REM
REM  Pred push na soukromy registry se musis prihlasit (docker login).
REM ============================================================

set "IMAGE_NAME=zoraxy-multi"
set "IMAGE_TAG=latest"

REM ---- TADY nastav svuj cil --------------------------------
set "REGISTRY=youruser/zoraxy-multi"
REM -----------------------------------------------------------

echo.
echo  ================================================
echo   Zoraxy - push image do registru
echo  ================================================
echo   Cil: %REGISTRY%:%IMAGE_TAG%
echo.

docker tag %IMAGE_NAME%:%IMAGE_TAG% %REGISTRY%:%IMAGE_TAG%
if errorlevel 1 (
    echo   CHYBA: docker tag selhal.
    pause
    exit /b 1
)

docker push %REGISTRY%:%IMAGE_TAG%
if errorlevel 1 (
    echo   CHYBA: docker push selhal. Jsi prihlaseny? (docker login)
    pause
    exit /b 1
)

echo.
echo  ================================================
echo   Hotovo! Na serveru pak:
echo     docker pull %REGISTRY%:%IMAGE_TAG%
echo  ================================================
echo.
pause
