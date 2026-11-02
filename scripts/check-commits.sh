#!/usr/bin/env bash
# Runs the commit-message rules over every commit in a revision range.
set -euo pipefail

range="${1:-HEAD~1..HEAD}"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

for commit in $(git rev-list "$range"); do
  git show -s --format=%B "$commit" > "$tmp"
  scripts/check-commit-msg.sh "$tmp" || {
    echo "  commit: $commit" >&2
    exit 1
  }
done
