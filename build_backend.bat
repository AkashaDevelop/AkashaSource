@echo off
REM ～用 garble 混淆编译后端二进制，每次构建随机种子，逆向成本拉满～
setlocal

set SCRIPT_DIR=%~dp0
set OUT_DIR=%SCRIPT_DIR%dist

if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

cd /d "%SCRIPT_DIR%backend"

REM 下载依赖（含 go.mod 中声明的 garble tool）
go mod tidy || exit /b 1

REM 先为 host 平台编译 garble（避免 GOOS/GOARCH 环境变量影响 garble 自身编译）
go build -o "%TEMP%\garble.exe" mvdan.cc/garble || exit /b 1

REM -literals  混淆字符串/数字常量字面量
REM -tiny      去除更多调试信息，进一步缩小体积
REM -seed=random  每次构建换一个随机混淆种子，杜绝跨版本比对符号
REM -trimpath  去除本地文件系统路径
REM -ldflags="-s -w"  去除符号表和 DWARF 调试信息
"%TEMP%\garble.exe" -literals -tiny -seed=random build -trimpath -ldflags="-s -w" -o "%OUT_DIR%\akasha.exe" . || exit /b 1

echo [宸汐御安全] 混淆后端二进制已编译到 %OUT_DIR%\akasha.exe
