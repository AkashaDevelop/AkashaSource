#!/usr/bin/env bash
# ～用 garble 混淆编译后端二进制，每次构建随机种子，逆向成本拉满～
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/dist"

mkdir -p "$OUT_DIR"

cd "$SCRIPT_DIR/backend"

# 下载依赖（含 go.mod 中声明的 garble tool）
go mod tidy

# -literals  混淆字符串/数字常量字面量
# -tiny      去除更多调试信息，进一步缩小体积
# -seed=random  每次构建换一个随机混淆种子，杜绝跨版本比对符号
# -trimpath  去除本地文件系统路径
# -ldflags="-s -w"  去除符号表和 DWARF 调试信息
go run mvdan.cc/garble -literals -tiny -seed=random build -trimpath -ldflags="-s -w" -o "$OUT_DIR/akasha" .

echo "[宸汐御安全] 混淆后端二进制已编译到 $OUT_DIR/akasha"
