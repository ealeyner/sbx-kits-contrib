# syntax=docker/dockerfile:1.7
# NanoClaw sandbox image.
#
# Everything needed to run NanoClaw is pre-baked so sandbox creation is
# fast: the kit (spec.yaml) only applies policy — network rules, credential
# proxying, published ports — and the entrypoint starts the pre-built
# service. This replaces the previous flow that cloned, pnpm-installed,
# tsc-built, and docker-built the agent image at first boot (5-12 minutes).
#
# Build with scripts/build-image.sh, which first builds/pulls the inner
# container images (nanoclaw-agent, OneCLI gateway, postgres) and stages
# them as images/<arch>/images.tar for the COPY near the bottom.

FROM docker/sandbox-templates:claude-code-docker

ARG NANOCLAW_REPO=https://github.com/nanocoai/nanoclaw.git
ARG NANOCLAW_REF=main
# Keep in lockstep with package.json `packageManager` in the nanoclaw repo.
ARG PNPM_VERSION=10.33.0

USER root
# Node 22 (nanoclaw requires >=20, pnpm v10 requires >=22) and pnpm.
RUN npm install -g n && n 22 && npm install -g "pnpm@${PNPM_VERSION}"
# OneCLI installs its CLI into ~/.local/bin; keep it on PATH in every shell
# context the sandbox runtime spawns.
RUN echo 'export PATH="/home/agent/.local/bin:$PATH"' >> /etc/sandbox-persistent.sh

USER agent
WORKDIR /home/agent
# Full clone (not --depth 1): nanoclaw's /add-<channel> skills fetch the
# upstream `channels` and `providers` branches at runtime.
RUN git clone "${NANOCLAW_REPO}" nanoclaw && \
    cd nanoclaw && \
    git checkout --quiet "${NANOCLAW_REF}" && \
    pnpm install --network-concurrency=1 && \
    pnpm rebuild better-sqlite3 && \
    pnpm run build

# Stamp nanoclaw's upgrade state: the image bake is the install (deps +
# build are done), and without the recorded version the service's upgrade
# tripwire refuses to start (docs/upgrade-recovery.md upstream).
RUN cd nanoclaw && pnpm exec tsx scripts/upgrade-state.ts set

# OneCLI CLI binary. The gateway itself (onecli + postgres containers) is
# seeded into the inner Docker daemon from images.tar at first boot.
RUN curl -fsSL https://onecli.sh/cli/install | sh && \
    /home/agent/.local/bin/onecli version

USER root
COPY --chmod=0755 scripts/start-nanoclaw.sh /usr/local/bin/nanoclaw-start
COPY --chmod=0755 scripts/load-images.sh /usr/local/bin/nanoclaw-load-images
# Launcher for manual invocation (sbx exec <sandbox> nanoclaw ...).
RUN printf '#!/bin/sh\ncd /home/agent/nanoclaw\nexec node dist/index.js "$@"\n' > /usr/local/bin/nanoclaw && \
    chmod 0755 /usr/local/bin/nanoclaw

# Pre-built inner images, loaded into the sandbox's Docker daemon on first
# boot by nanoclaw-load-images. Staged per-arch by scripts/build-image.sh.
# Create the parent explicitly: COPY --chmod would otherwise apply the
# file mode (no execute bit) to the directory it implicitly creates,
# making it untraversable.
RUN install -d -m 0755 /opt/nanoclaw
ARG TARGETARCH
COPY --chmod=0644 images/${TARGETARCH}/images.tar /opt/nanoclaw/images.tar

USER agent
WORKDIR /home/agent
CMD ["/usr/local/bin/nanoclaw-start"]
