@echo off
setlocal
chcp 65001 >nul
title Zoraxy - build docker image

REM ============================================================
REM  build.bat
REM
REM  Postavi docker image z tveho upraveneho src/ a ulozi ho jako
REM  tar do build/output/zoraxy-multi.tar. Build bezi pres WSL
REM  (protoze tam mas docker), takze musis mit spusteny WSL + Docker.
REM
REM  Vysledny tar pak na cili nahrajes:  docker load -i zoraxy-multi.tar
REM  (nebo ho pushnes do registru, viz push.bat)
REM ============================================================

REM ---- nastaveni ----------------------------------------------
set "IMAGE_NAME=zoraxy-multi"
set "IMAGE_TAG=latest"

REM absolutni cesta ke koreni repozitare (slozka build je v koreni)
set "REPO=%~dp0.."
for %%I in ("%REPO%") do set "REPO=%%~fI"

echo.
echo  ================================================
echo   Zoraxy - build docker image (pres WSL)
echo  ================================================
echo   Repo : %REPO%
echo   Image: %IMAGE_NAME%:%IMAGE_TAG%
echo.

REM ---- 1) priprava build contextu -----------------------------
echo  [1/3] Pripravuju build context (build\.context)...
wsl --cd "%REPO%" -u root -- bash -lc "rm -rf build/.context && mkdir -p build/.context && cp -r src build/.context/src && cp docker/Dockerfile docker/entrypoint.go build/.context/"
if errorlevel 1 (
    echo   CHYBA: nepodarilo se pripravit build context.
    pause
    exit /b 1
)

REM ---- 2) docker build ---------------------------------------
echo  [2/3] Stavim image (docker build - muze to chvili trvat)...
wsl --cd "%REPO%" -u root -- bash -lc "docker build -t %IMAGE_NAME%:%IMAGE_TAG% build/.context"
if errorlevel 1 (
    echo   CHYBA: docker build selhal. Podivej se na log vyse.
    pause
    exit /b 1
)

REM ---- 3) docker save do output ------------------------------
echo  [3/3] Ukladam image do build\output...
wsl --cd "%REPO%" -u root -- bash -lc "mkdir -p build/output && docker save %IMAGE_NAME%:%IMAGE_TAG% -o build/output/%IMAGE_NAME%.tar && ls -lh build/output/%IMAGE_NAME%.tar"
if errorlevel 1 (
    echo   CHYBA: docker save selhal.
    pause
    exit /b 1
)

echo.
echo  ================================================
echo   Hotovo!
echo   Tar je tady: build\output\%IMAGE_NAME%.tar
echo.
echo   Na cili ho nahrajes:  docker load -i %IMAGE_NAME%.tar
echo  ================================================
echo.
pause
