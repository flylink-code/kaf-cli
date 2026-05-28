#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export GOPROXY=https://goproxy.cn,direct
LDFLAGS="-s -w -X main.version=dev"

mkdir -p build/windows-amd64 build/windows-386 build/linux-amd64 build/darwin-amd64

build() {
  local out=$1 pkg=$2 extra=${3:-}
  echo ">> building ${out} ..."
  if [ -n "$extra" ]; then
    go build -ldflags "${LDFLAGS} ${extra}" -o "$out" "$pkg"
  else
    go build -ldflags "$LDFLAGS" -o "$out" "$pkg"
  fi
  echo "   ok"
}

GOOS=windows GOARCH=amd64 build build/windows-amd64/kaf-cli.exe ./cmd
GOOS=windows GOARCH=amd64 build build/windows-amd64/kaf-cli-gui.exe ./cmd/gui "-H windowsgui"
GOOS=windows GOARCH=386  go build -ldflags "$LDFLAGS" -o build/windows-386/kaf-cli.exe ./cmd
GOOS=linux   GOARCH=amd64 go build -ldflags "$LDFLAGS" -o build/linux-amd64/kaf-cli ./cmd
GOOS=darwin  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o build/darwin-amd64/kaf-cli ./cmd

echo ""
echo "build done!"
echo "  build/windows-amd64/kaf-cli.exe"
echo "  build/windows-amd64/kaf-cli-gui.exe"
