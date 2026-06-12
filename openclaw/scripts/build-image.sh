#!/usr/bin/env bash
# Build the OpenClaw sandbox image. No inner container images to stage —
# OpenClaw runs everything in-process by default — so this is a plain
# docker build with pinned versions.
#
# Overridable via env:
#   OPENCLAW_VERSION  openclaw npm version to bake (date-based upstream)
#   IMAGE             output tag for the sandbox image
set -euo pipefail

KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPENCLAW_VERSION="${OPENCLAW_VERSION:-2026.6.5}"
IMAGE="${IMAGE:-openclaw-sbx:local}"

echo "==> Building sandbox image ${IMAGE} (openclaw@${OPENCLAW_VERSION})"
docker build -t "$IMAGE" \
    --build-arg OPENCLAW_VERSION="$OPENCLAW_VERSION" \
    "$KIT_DIR"

echo "==> Done: ${IMAGE}"
