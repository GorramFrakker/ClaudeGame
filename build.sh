#!/usr/bin/env bash
#
# Build STARFALL. Produces a standalone Windows executable that needs no
# installer, runtime, or asset files — everything is embedded in the .exe.
#
# Works from any OS that has the Go toolchain (1.24+). No C compiler required
# for the Windows target.
#
#   ./build.sh            # build Windows amd64 -> dist/STARFALL.exe
#   ./build.sh all        # also build Windows arm64
#
set -euo pipefail
cd "$(dirname "$0")"

OUT=dist
mkdir -p "$OUT"

LDFLAGS="-s -w -H windowsgui"   # strip symbols; GUI subsystem = no console window

build() {
  local goos=$1 goarch=$2 name=$3
  echo ">> ${goos}/${goarch} -> ${OUT}/${name}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "$LDFLAGS" -o "${OUT}/${name}" .
}

build windows amd64 STARFALL.exe

if [[ "${1:-}" == "all" ]]; then
  build windows arm64 STARFALL-arm64.exe
fi

echo
echo "Done. Run the game on Windows by double-clicking ${OUT}/STARFALL.exe"
ls -lh "$OUT"
