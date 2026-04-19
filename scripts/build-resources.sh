#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_DIR="$ROOT_DIR/go"
RESOURCES_DIR="$ROOT_DIR/resources/bin"
DIST_DIR="$ROOT_DIR/dist/package/bin"

TARGETS=(
  "darwin arm64"
)

map_arch_dir() {
  case "$1" in
    amd64) echo "x64" ;;
    arm64) echo "arm64" ;;
    *)
      echo "Unsupported arch dir mapping: $1" >&2
      exit 1
      ;;
  esac
}

binary_name() {
  case "$1" in
    windows) echo "pi-memory-backend.exe" ;;
    *) echo "pi-memory-backend" ;;
  esac
}

mkdir -p "$RESOURCES_DIR" "$DIST_DIR"

for target in "${TARGETS[@]}"; do
  read -r goos goarch <<< "$target"
  arch_dir="$(map_arch_dir "$goarch")"
  target_dir="$RESOURCES_DIR/${goos}-${arch_dir}"
  out_name="$(binary_name "$goos")"
  out_path="$target_dir/$out_name"

  mkdir -p "$target_dir"

  echo "Building resource binary for $goos/$goarch -> $out_path"
  (
    cd "$GO_DIR"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -o "$out_path" ./cmd/pi-memory-backend
  )

  chmod +x "$out_path" 2>/dev/null || true

done

local_goos="$(go env GOOS)"
local_goarch="$(go env GOARCH)"
local_out="$DIST_DIR/$(binary_name "$local_goos")"

echo "Building local development binary for $local_goos/$local_goarch -> $local_out"
(
  cd "$GO_DIR"
  CGO_ENABLED=0 GOOS="$local_goos" GOARCH="$local_goarch" go build -o "$local_out" ./cmd/pi-memory-backend
)

chmod +x "$local_out" 2>/dev/null || true

echo "Built packaged resource binaries under $RESOURCES_DIR"
echo "Built local development binary under $DIST_DIR"
