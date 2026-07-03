#!/usr/bin/env bash
# ～编译 宸汐御安全 WASM 运行时，生成 cx_runtime.wasm～
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/../frontend/src/lib/cxsec"

mkdir -p "$OUT_DIR"

cd "$SCRIPT_DIR/wasm"

# 下载依赖
go mod tidy

# 编译 WASM（关闭调试符号，减小体积）
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUT_DIR/cx_runtime.wasm" .

# 拷贝 Go 提供的 wasm_exec.js 胶水文件
GOROOT=$(go env GOROOT)
cp "$GOROOT/misc/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"

echo "[宸汐御安全] WASM 已编译到 $OUT_DIR/cx_runtime.wasm"
echo "[宸汐御安全] wasm_exec.js 已拷贝到 $OUT_DIR/wasm_exec.js"
