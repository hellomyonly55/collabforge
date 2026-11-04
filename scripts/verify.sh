#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
command -v go >/dev/null 2>&1 || { echo '需要 Go 1.26+；请安装并加入 PATH。' >&2; exit 1; }
command -v node >/dev/null 2>&1 || { echo '需要 Node.js 22.12+。' >&2; exit 1; }
[ -d frontend/node_modules ] || { echo '请先在 frontend 执行 npm ci。' >&2; exit 1; }
node --test scripts/init-env.test.mjs
docker compose config --quiet
(cd backend && go test ./... && go vet ./... && go build ./cmd/server ./cmd/migrate ./cmd/admin ./cmd/checkdocs && go run ./cmd/checkdocs)
(cd frontend && npm run build)
cleanup() { docker compose --profile test stop mysql-test redis-test; }
trap cleanup EXIT
./scripts/test-integration.sh
./scripts/test-e2e.sh
./scripts/test-workflow.sh
cleanup
trap - EXIT
printf '%s\n' 'PASS: TaskFlow release verification completed.'
