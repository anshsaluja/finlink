#!/usr/bin/env sh
set -eu

DATABASE_URL="${FINLINK_DATABASE_URL:-postgres://finlink:finlink@localhost:5432/finlink?sslmode=disable}"

for migration in migrations/*.up.sql; do
  echo "applying ${migration}"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
done
