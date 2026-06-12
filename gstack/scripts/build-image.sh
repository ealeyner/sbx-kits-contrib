#!/usr/bin/env bash
# Build the gstack sandbox image — a plain docker build with pinned refs
# (no inner container images; the browse daemon runs in-process).
#
# Overridable via env:
#   GSTACK_REF   git commit SHA to bake (gstack publishes no release tags)
#   BUN_VERSION  Bun version (upstream pins 1.3.10)
#   IMAGE        output tag for the sandbox image
set -euo pipefail

KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GSTACK_REF="${GSTACK_REF:-a5833c413f98b13f105beac96262e8098b628461}"
BUN_VERSION="${BUN_VERSION:-1.3.10}"
IMAGE="${IMAGE:-gstack-sbx:local}"

echo "==> Building sandbox image ${IMAGE} (gstack @ ${GSTACK_REF})"
docker build -t "$IMAGE" \
    --build-arg GSTACK_REF="$GSTACK_REF" \
    --build-arg BUN_VERSION="$BUN_VERSION" \
    "$KIT_DIR"

echo "==> Done: ${IMAGE}"
