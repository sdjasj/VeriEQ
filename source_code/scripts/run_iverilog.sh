#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
mkdir -p .gocache

THREADS="${1:-64}"
COUNT="${2:-2}"
CONFIG_PATH="${3:-$ROOT_DIR/config.json}"

GOCACHE="$ROOT_DIR/.gocache" go run . -fuzzer iverilog -threads "$THREADS" -count "$COUNT" -config "$CONFIG_PATH"
