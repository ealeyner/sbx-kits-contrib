#!/usr/bin/env bash
# Build the NanoClaw sandbox image:
#
#   1. clone nanoclaw at the pinned ref
#   2. build the nanoclaw-agent container image from its container/ dir
#   3. pull the OneCLI gateway + postgres images
#   4. docker save all three into images/<arch>/images.tar
#   5. docker build the sandbox image with the tar embedded
#
# Overridable via env:
#   NANOCLAW_REPO   upstream repo URL
#   NANOCLAW_REF    git ref to bake (pin a commit for reproducible builds)
#   ONECLI_VERSION  OneCLI gateway version — keep in lockstep with
#                   ONECLI_GATEWAY_VERSION in nanoclaw's setup/onecli.ts
#   IMAGE           output tag for the sandbox image
set -euo pipefail

KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NANOCLAW_REPO="${NANOCLAW_REPO:-https://github.com/nanocoai/nanoclaw.git}"
NANOCLAW_REF="${NANOCLAW_REF:-36cbf17e107fd0f8daea4ceb2ac523d9f0d88915}"
ONECLI_VERSION="${ONECLI_VERSION:-1.23.0}"
IMAGE="${IMAGE:-nanoclaw-sbx:local}"
ARCH="${ARCH:-$(docker version --format '{{.Server.Arch}}')}"

workdir="$(mktemp -d /tmp/nanoclaw-image.XXXXXX)"
trap 'rm -rf "$workdir"' EXIT

echo "==> Cloning nanoclaw @ ${NANOCLAW_REF}"
git clone "$NANOCLAW_REPO" "$workdir/nanoclaw"
git -C "$workdir/nanoclaw" checkout --quiet "$NANOCLAW_REF"

# Switch the agent image's apt sources to HTTPS before building. Some
# networks (notably corporate ones with TLS-inspecting appliances) block
# apt's plain-HTTP traffic; HTTPS works everywhere deb.debian.org does.
# node:22-slim ships no CA bundle (ca-certificates is only installed by
# the apt step itself), so borrow alpine's bundle for the bootstrap —
# the real ca-certificates package overwrites it moments later.
agent_dockerfile="$workdir/nanoclaw/container/Dockerfile"
ca_copy='COPY --from=alpine:3 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt'
apt_https='RUN sed -i "s|http://deb.debian.org|https://deb.debian.org|g" /etc/apt/sources.list.d/debian.sources'
awk -v l1="$ca_copy" -v l2="$apt_https" '{print} /^FROM /{print l1; print l2}' \
    "$agent_dockerfile" > "$agent_dockerfile.tmp"
mv "$agent_dockerfile.tmp" "$agent_dockerfile"
grep -q 'https://deb.debian.org' "$agent_dockerfile" || {
    echo "error: failed to patch apt sources in $agent_dockerfile" >&2
    exit 1
}

echo "==> Building nanoclaw-agent image"
docker build -t nanoclaw-agent:latest "$workdir/nanoclaw/container"

echo "==> Pulling OneCLI gateway images"
docker pull "ghcr.io/onecli/onecli:${ONECLI_VERSION}"
docker pull postgres:18-alpine

echo "==> Saving inner images to images/${ARCH}/images.tar"
mkdir -p "$KIT_DIR/images/$ARCH"
docker save -o "$KIT_DIR/images/$ARCH/images.tar" \
    nanoclaw-agent:latest \
    "ghcr.io/onecli/onecli:${ONECLI_VERSION}" \
    postgres:18-alpine

echo "==> Building sandbox image ${IMAGE}"
docker build -t "$IMAGE" \
    --build-arg NANOCLAW_REPO="$NANOCLAW_REPO" \
    --build-arg NANOCLAW_REF="$NANOCLAW_REF" \
    "$KIT_DIR"

echo "==> Done: ${IMAGE}"
