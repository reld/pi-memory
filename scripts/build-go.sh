#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_DIR="$ROOT_DIR/go"
OUT_DIR="$ROOT_DIR/dist/package/bin"

mkdir -p "$OUT_DIR"

(
  cd "$GO_DIR"
  go build -o "$OUT_DIR/pi-memory-backend" ./cmd/pi-memory-backend
)

echo "Built Go backend to $OUT_DIR/pi-memory-backend"
