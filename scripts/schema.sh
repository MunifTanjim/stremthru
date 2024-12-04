#!/usr/bin/env bash

set -euo pipefail

source .env

declare -r cmd="${1:-"help"}"
shift || true

declare db_url="${STREMTHRU_DATABASE_URL}"
db_url="${db_url/file:/sqlite://}"

case "${cmd}" in
inspect)
  atlas schema inspect -u "${db_url}"
  ;;
migrate)
  atlas schema apply -u "${db_url}" --file schema.hcl
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
