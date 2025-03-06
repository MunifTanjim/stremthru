#!/usr/bin/env bash

set -euo pipefail

export GOOSE_DBSTRING="${STREMTHRU_DATABASE_URI}"

declare -a args=("-table" "db_migration_version")

case "${GOOSE_DBSTRING}" in
sqlite://*)
  export GOOSE_DBSTRING="${GOOSE_DBSTRING#sqlite://}"
  export GOOSE_DRIVER="sqlite"
  export GOOSE_MIGRATION_DIR="migrations/sqlite"
  ;;
postgresql://*)
  export GOOSE_DRIVER="postgres"
  export GOOSE_MIGRATION_DIR="migrations/postgres"
  ;;
esac

goose ${args[@]} ${@}
