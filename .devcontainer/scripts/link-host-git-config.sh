#!/usr/bin/env bash
set -euo pipefail

HOST_GITCONFIG="/host-home/.gitconfig"

if [[ ! -f "$HOST_GITCONFIG" ]]; then
    exit 0
fi

if ! git config --global --get-all include.path 2>/dev/null | grep -Fxq "$HOST_GITCONFIG"; then
    git config --global --add include.path "$HOST_GITCONFIG"
fi
