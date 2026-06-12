#!/bin/sh
# Seed the sandbox's inner Docker daemon with the pre-built images embedded
# in this sandbox image at /opt/nanoclaw/images.tar (nanoclaw-agent, OneCLI
# gateway, postgres). Loading from the local archive takes ~30s on first
# boot and replaces the 5-12 minute in-sandbox build/pull the kit needed
# before. Subsequent boots no-op because the images are already present.
#
# Safe to call concurrently: one caller loads, the rest wait on the lock.

ARCHIVE=/opt/nanoclaw/images.tar
LOCK=/tmp/nanoclaw-load-images.lock

[ -f "$ARCHIVE" ] || exit 0

i=0
until docker info >/dev/null 2>&1; do
    sleep 1
    i=$((i+1))
    if [ $i -ge 60 ]; then
        echo "Docker daemon not ready after 60s" >&2
        exit 1
    fi
done

docker image inspect nanoclaw-agent:latest >/dev/null 2>&1 && exit 0

if mkdir "$LOCK" 2>/dev/null; then
    trap 'rmdir "$LOCK" 2>/dev/null' EXIT INT TERM
    echo "First boot: seeding inner Docker with pre-built images (~30s)..." >&2
    docker load -i "$ARCHIVE" >&2
else
    echo "Waiting for image load started by another process..." >&2
    i=0
    while [ -d "$LOCK" ]; do
        sleep 2
        i=$((i+1))
        if [ $i -ge 300 ]; then
            echo "Timed out waiting for image load" >&2
            exit 1
        fi
    done
    docker image inspect nanoclaw-agent:latest >/dev/null 2>&1 || exit 1
fi
