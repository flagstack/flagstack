#!/usr/bin/env bash
set -Eeuo pipefail

error_trap() {
    local code=$?
    echo "post-create failed at line $LINENO: ${BASH_COMMAND} (exit $code)" >&2
    exit "$code"
}
trap error_trap ERR

cd /workspace

corepack enable
corepack prepare pnpm@11.22.0 --activate

echo "Installing frontend dependencies..."
pnpm --dir frontend install

echo "Downloading Go modules..."
(
    cd backend
    go mod download
    go generate ./ent
)

git config --global --add safe.directory /workspace

bash .devcontainer/scripts/setup-shell.sh
bash .devcontainer/scripts/link-host-git-config.sh

echo "FlagStack development environment is ready."
