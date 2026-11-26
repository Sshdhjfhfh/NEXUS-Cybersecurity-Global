#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p build
version="$(awk '$1 == "go" {print $2; exit}' go.mod)"
compiler="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go${version}.linux-amd64/bin/go"
if [[ ! -x "$compiler" ]]; then
  compiler="$(command -v go || true)"
fi
if [[ -z "$compiler" ]]; then
  if [[ -x build/nexus-fair ]]; then
    exec build/nexus-fair "$@"
  fi
  printf 'Falta Go o el ejecutable preparado. Consulta docs/FERIA_NEXUS.md.\n' >&2
  exit 1
fi
# Dependencies must be prepared before the exhibition. Never download at the stand.
GOPROXY=off GOTOOLCHAIN=local "$compiler" build -trimpath -o build/nexus-fair ./cmd/nexus-fair
exec build/nexus-fair "$@"
