#!/usr/bin/env bash
# Build the Paperclip sandbox image. No inner container images to stage —
# paperclip runs agent CLIs as child processes in the same container — so
# this is a plain docker build with a pinned version.
#
# Overridable via env:
#   PAPERCLIP_VERSION  paperclipai npm version to bake (calendar versioning)
#   IMAGE              output tag for the sandbox image
set -euo pipefail

KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PAPERCLIP_VERSION="${PAPERCLIP_VERSION:-2026.609.0}"
IMAGE="${IMAGE:-paperclip-sbx:local}"

echo "==> Building sandbox image ${IMAGE} (paperclipai@${PAPERCLIP_VERSION})"
docker build -t "$IMAGE" \
    --build-arg PAPERCLIP_VERSION="$PAPERCLIP_VERSION" \
    "$KIT_DIR"

echo "==> Done: ${IMAGE}"
