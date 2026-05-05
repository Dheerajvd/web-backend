#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

case "${1:-}" in
  generate-swagger)
    echo "Generating swagger docs..."
    # No global swag CLI required.
    go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/main.go -o docs --parseDependency --parseInternal
    ;;

  run)
    echo "Starting server..."
    go run ./cmd/main.go
    ;;

  *)
    echo "Usage:"
    echo "  ./run.sh generate-swagger   # generate ./docs (swagger)"
    echo "  ./run.sh run                # run the server"
    exit 1
    ;;
esac

