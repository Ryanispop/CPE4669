#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p results
go build -o ./lab7 .
case "${1:-demo}" in
  demo) ./lab7 -mode demo | tee results/demo.txt ;;
  bench) ./lab7 -mode bench -workload "${2:-1000}" -repeats "${3:-3}" | tee results/benchmark.txt ;;
  test) go test -race ./...; go vet ./... ;;
  *) echo "Usage: ./run.sh [demo|bench [keys] [repeats]|test]" >&2; exit 1 ;;
esac
