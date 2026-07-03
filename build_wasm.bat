@echo off
REM 宸汐御安全 WASM 构建脚本 (Windows batch)
setlocal

set OUT_DIR=%~dp0frontend\src\lib\cxsec
if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

cd /d "%~dp0wasm"

echo [宸汐御安全] 安装依赖...
go mod tidy

echo [宸汐御安全] 编译 WASM...
set GOOS=js
set GOARCH=wasm
go build -ldflags="-s -w" -o "%OUT_DIR%\cx_runtime.wasm" .

if errorlevel 1 (
    echo [错误] WASM 编译失败
    exit /b 1
)

echo [宸汐御安全] 拷贝 wasm_exec.js...
for /f "delims=" %%i in ('go env GOROOT') do set GOROOT=%%i
copy /y "%GOROOT%\misc\wasm\wasm_exec.js" "%OUT_DIR%\wasm_exec.js"

echo [宸汐御安全] 完成！输出目录: %OUT_DIR%
endlocal
