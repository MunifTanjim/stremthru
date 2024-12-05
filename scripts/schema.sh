#!/usr/bin/env bash

set -euo pipefail

declare -r cmd="${1:-"help"}"
shift || true

declare db_url="${STREMTHRU_DATABASE_URI}"

declare schema_file="schema.hcl"

case "${db_url}" in
postgresql://*)
  schema_file="schema.postgres.hcl"
  ;;
esac

case "${cmd}" in
inspect)
  atlas schema inspect -u "${db_url}"
  ;;
migrate)
  atlas schema apply -u "${db_url}" --file "${schema_file}"
  ;;
help)
  cat <<EOF
./scripts/schema.sh <COMMAND>

COMMANDS:
  inspect
  migrate
EOF
  ;;
*)
  exit 1
  ;;
esac
