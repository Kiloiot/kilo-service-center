#!/usr/bin/env bash
# Compares the public protobuf contract with the latest release tag and fails on
# any breaking change that is not listed in breaking-exceptions.txt. A finding
# is matched by its file and text, without buf's line:column position, so an
# edit elsewhere in a proto file never invalidates an exception. Each exception
# must be sanctioned by a CHANGELOG entry; a listed exception that buf no longer
# reports is stale and fails the check too.
set -euo pipefail

PROTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GIT_ROOT="$(git -C "$PROTO_DIR" rev-parse --show-toplevel)"
EXCEPTIONS="$PROTO_DIR/breaking-exceptions.txt"

# The proto directory sits at KC-Core/api/proto in the public repository and one
# level deeper in the workspace that exports it; git reports the difference.
SUBDIR="$(git -C "$PROTO_DIR" rev-parse --show-prefix)"
SUBDIR="${SUBDIR%/}"
TAG_SUBDIR="${SUBDIR#kilocenter-modules/}"

TAG="${PROTO_BREAKING_AGAINST:-$(git -C "$GIT_ROOT" tag -l 'v*' | sort -V | tail -n 1)}"
if [[ -z "$TAG" ]]; then
  echo "check-breaking: no release tag found to compare against" >&2
  exit 1
fi

reported="$(cd "$PROTO_DIR" && buf breaking --against "$GIT_ROOT/.git#tag=$TAG,subdir=$TAG_SUBDIR" 2>&1 | sed "s#^$PROTO_DIR/##" || true)"
reported="$(printf '%s\n' "$reported" | sed -E '/^$/d; s/^([^:]+):[0-9]+:[0-9]+:/\1:/')"

expected=""
if [[ -f "$EXCEPTIONS" ]]; then
  expected="$(grep -v '^[[:space:]]*#' "$EXCEPTIONS" | sed '/^[[:space:]]*$/d' || true)"
fi

unlisted="$(comm -23 <(printf '%s\n' "$reported" | sed '/^$/d' | sort -u) <(printf '%s\n' "$expected" | sed '/^$/d' | sort -u))"
stale="$(comm -13 <(printf '%s\n' "$reported" | sed '/^$/d' | sort -u) <(printf '%s\n' "$expected" | sed '/^$/d' | sort -u))"

status=0
if [[ -n "$unlisted" ]]; then
  echo "check-breaking: breaking changes against $TAG that no CHANGELOG entry sanctions:" >&2
  printf '%s\n' "$unlisted" | sed 's/^/  /' >&2
  status=1
fi
if [[ -n "$stale" ]]; then
  echo "check-breaking: stale exceptions (buf no longer reports them against $TAG); remove them:" >&2
  printf '%s\n' "$stale" | sed 's/^/  /' >&2
  status=1
fi
if [[ $status -eq 0 ]]; then
  count="$(printf '%s\n' "$expected" | sed '/^$/d' | wc -l | tr -d ' ')"
  echo "check-breaking: contract compatible with $TAG ($count sanctioned exception(s))"
fi
exit $status
