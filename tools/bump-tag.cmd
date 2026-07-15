@echo off
setlocal EnableExtensions EnableDelayedExpansion
REM bump-tag.cmd — Windows CMD：打 annotated tag 并推送
REM 用法:
REM   tools\bump-tag.cmd              拉取最新 v*，patch +1 后打 tag
REM   tools\bump-tag.cmd v1.0.3       直接打指定 tag，不拉取远端最新 tag
REM 优先调用同目录 bump-tag.sh（Git Bash）；否则使用纯 CMD 实现

set "SCRIPT_DIR=%~dp0"
set "ROOT=%SCRIPT_DIR%.."
pushd "%ROOT%" >nul || (
  echo [ERROR] cannot enter repo root
  exit /b 1
)

set "REMOTE=origin"
if defined REMOTE_ENV set "REMOTE=%REMOTE_ENV%"
set "TAG_PREFIX=FlashDock"
set "EXPLICIT_TAG=%~1"

where bash >nul 2>&1
if not errorlevel 1 (
  if defined EXPLICIT_TAG (
    bash "%SCRIPT_DIR%bump-tag.sh" "!EXPLICIT_TAG!"
  ) else (
    bash "%SCRIPT_DIR%bump-tag.sh"
  )
  set "ERR=!ERRORLEVEL!"
  popd >nul
  exit /b !ERR!
)

where git >nul 2>&1
if errorlevel 1 (
  echo [ERROR] git not found in PATH
  popd >nul
  exit /b 1
)

git rev-parse --is-inside-work-tree >nul 2>&1
if errorlevel 1 (
  echo [ERROR] not a git repository
  popd >nul
  exit /b 1
)

if defined EXPLICIT_TAG (
  set "RAW=!EXPLICIT_TAG!"
  echo !RAW!| findstr /r "^v" >nul
  if errorlevel 1 set "RAW=v!RAW!"
  echo !RAW!| findstr /r "^v[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$" >nul
  if errorlevel 1 (
    echo [ERROR] invalid tag: !EXPLICIT_TAG! ^(expect v1.2.3^)
    popd >nul
    exit /b 1
  )
  set "NEW_TAG=!RAW!"
  echo [INFO] use explicit tag: !NEW_TAG! ^(skip fetch latest^)
  goto :have_tag
)

echo [INFO] fetch tags from %REMOTE% ...
git fetch --tags --force %REMOTE%
if errorlevel 1 (
  echo [ERROR] git fetch failed
  popd >nul
  exit /b 1
)

set "LATEST="
for /f "usebackq delims=" %%T in (`git tag -l "v[0-9]*" --sort=-v:refname`) do (
  if not defined LATEST set "LATEST=%%T"
)

if not defined LATEST (
  echo [WARN] no v* tag found, start from v1.0.0
  set "NEW_TAG=v1.0.0"
  goto :have_tag
)

echo [INFO] latest tag: !LATEST!
set "VER=!LATEST:~1!"
for /f "tokens=1,2,3 delims=.+-" %%A in ("!VER!") do (
  set "MAJOR=%%A"
  set "MINOR=%%B"
  set "PATCH=%%C"
)

echo !MAJOR!| findstr /r "^[0-9][0-9]*$" >nul || goto :bad_ver
echo !MINOR!| findstr /r "^[0-9][0-9]*$" >nul || goto :bad_ver
echo !PATCH!| findstr /r "^[0-9][0-9]*$" >nul || goto :bad_ver

set /a NEXT_PATCH=!PATCH!+1
set "NEW_TAG=v!MAJOR!.!MINOR!.!NEXT_PATCH!"
goto :have_tag

:bad_ver
echo [ERROR] cannot parse version: !LATEST! ^(expect v1.2.3^)
popd >nul
exit /b 1

:have_tag
git rev-parse -q --verify "refs/tags/!NEW_TAG!" >nul 2>&1
if not errorlevel 1 (
  echo [ERROR] local tag already exists: !NEW_TAG!
  popd >nul
  exit /b 1
)

git ls-remote --tags --exit-code %REMOTE% "refs/tags/!NEW_TAG!" >nul 2>&1
if not errorlevel 1 (
  echo [ERROR] remote tag already exists: !NEW_TAG!
  popd >nul
  exit /b 1
)

set "MSG=%TAG_PREFIX% !NEW_TAG!"
echo [INFO] create tag: !NEW_TAG!
echo [INFO] message: !MSG!
git tag -a "!NEW_TAG!" -m "!MSG!"
if errorlevel 1 (
  echo [ERROR] git tag failed
  popd >nul
  exit /b 1
)

echo [INFO] push tag to %REMOTE% ...
git push %REMOTE% "refs/tags/!NEW_TAG!"
if errorlevel 1 (
  echo [ERROR] git push failed
  popd >nul
  exit /b 1
)

echo [OK] done: !NEW_TAG!
popd >nul
exit /b 0
