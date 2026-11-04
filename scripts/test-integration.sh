#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ ! -f .env ]; then echo 'Create a trusted .env from .env.example first.' >&2; exit 1; fi
set -a
. ./.env
set +a
docker compose --profile test up -d --wait mysql-test redis-test
export MYSQL_HOST=127.0.0.1
export MYSQL_PORT="${TEST_MYSQL_PORT:-13306}"
export MYSQL_DATABASE=taskflow_test
export REDIS_ADDR="127.0.0.1:${TEST_REDIS_PORT:-16379}"
export TASKFLOW_INTEGRATION=1
cd backend
go test -count=1 -v ./tests/integration
