#!/usr/bin/env bash
# ～编译 宸汐御安全 WASM 运行时，生成 cx_runtime.wasm～
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/frontend/public/lib"

mkdir -p "$OUT_DIR"

cd "$SCRIPT_DIR/wasm"

# 下载依赖
go mod tidy

# 编译 WASM（关闭调试符号，减小体积）
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUT_DIR/cx_runtime.wasm" .

# 拷贝 Go 提供的 wasm_exec.js 胶水文件（Go 1.24+ 位于 lib/wasm/，更早在 misc/wasm/）
GOROOT=$(go env GOROOT)
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
  cp "$GOROOT/lib/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
else
  cp "$GOROOT/misc/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
fi

echo "[宸汐御安全] WASM 已编译到 $OUT_DIR/cx_runtime.wasm"
echo "[宸汐御安全] wasm_exec.js 已拷贝到 $OUT_DIR/wasm_exec.js"
