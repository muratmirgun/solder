#!/usr/bin/env bash
# Prints the stripped binary size of the same app built with solder and fx.
set -euo pipefail
cd "$(dirname "$0")"
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
for name in solder-app fx-app; do
  go build -ldflags="-s -w" -o "$out/$name" "./cmd/$name"
  printf '%-11s %6d KiB\n' "$name" $(( $(wc -c < "$out/$name") / 1024 ))
done
