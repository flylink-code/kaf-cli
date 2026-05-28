#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

export GOPROXY=https://goproxy.cn,direct
LDFLAGS="-s -w -X main.version=dev"

build() {
  local name=$1 pkg=$2 extra=${3:-}
  echo ">> building ${name} ..."
  if [ -n "$extra" ]; then
    GOOS=windows go build -ldflags "${LDFLAGS} ${extra}" -o "$name" "$pkg"
  else
    GOOS=windows go build -ldflags "$LDFLAGS" -o "$name" "$pkg"
  fi
  echo "   ok"
}

build kaf-cli.exe ./cmd
build kaf-cli-gui.exe ./cmd/gui "-H windowsgui"
GOOS=windows GOARCH=386 go build -ldflags "$LDFLAGS" -o kaf-cli_32.exe ./cmd
GOOS=linux   go build -ldflags "$LDFLAGS" -o kaf-cli-linux ./cmd
GOOS=darwin  go build -ldflags "$LDFLAGS" -o kaf-cli-darwin ./cmd

echo ""
echo "build done!"
