# gstack

A standalone sandbox kit (`kind: sandbox`, the v2 spec naming) for
[gstack](https://github.com/garrytan/gstack) — Garry Tan's Claude Code
skill pack: 23 opinionated slash commands (`/ship`, `/review`, `/qa`,
`/browse`, `/office-hours`, …) plus compiled Bun binaries including a
headless-Chromium browse daemon.

The kit uses a **pre-baked sandbox image** on the `claude-code` template:
Bun, the gstack checkout at a pinned commit with `./setup` already run
(all skills registered under `~/.claude/skills`), and Chromium for the
browse daemon. Attaching drops straight into a Claude Code session with
every skill available — nothing installs at sandbox creation.

## Usage

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=gstack" gstack
```

Then use the skills as usual: `/review`, `/qa <url>`, `/browse`, `/ship`,
… The browse daemon self-starts on first use (loopback only, random port,
per-project state in `<workspace>/.gstack/`).

## Ports

None published — the browse and design daemons bind loopback on random
ports and are only used in-sandbox.

## How auth works

Standard Anthropic wiring for Claude Code: `serviceDomains`/`serviceAuth`
for API-key auth and the OAuth flow against `platform.claude.com` with
proxy-managed sentinels. Set `ANTHROPIC_API_KEY` on the host to skip
OAuth.

The browse security stack's prompt-injection classifier lazy-loads from
`huggingface.co` on first `/browse` use (~112 MB, allowed in the network
policy). gstack's Supabase telemetry endpoint is deliberately not
allow-listed; its sync exits silently when unreachable.

## Image architecture

```
docker.io/ealeyner/gstack-sbx
└── FROM docker/sandbox-templates:claude-code
    ├── Bun 1.3.10 (/usr/local)
    ├── /opt/playwright-browsers        Chromium + xvfb/fonts for /browse
    └── ~/.claude/skills/gstack         checkout @ pinned SHA, ./setup run:
        ├── compiled binaries (browse, pdf, design, ...)
        ├── ~/.claude/skills/<name>/    all skills registered (symlinks)
        └── ~/.gstack/                  global state
```

gstack publishes no release tags — the image pins a commit SHA
(`GSTACK_REF` in `scripts/build-image.sh`). The checkout keeps `.git` so
`/gstack-upgrade` and version checks work from inside the sandbox.

## Building and publishing the image

```console
$ IMAGE=docker.io/<you>/gstack-sbx:latest ./scripts/build-image.sh
$ docker push docker.io/<you>/gstack-sbx:latest
```

To bump gstack: set `GSTACK_REF` to a new upstream commit SHA and rebuild.

## Debugging

```console
$ sbx exec <sandbox> -- ls /home/agent/.claude/skills/        # skills registered?
$ sbx exec <sandbox> -- /home/agent/.claude/skills/gstack/browse/dist/browse goto https://example.com
$ sbx exec <sandbox> -- cat <workspace>/.gstack/browse.json   # daemon port + token
```

See [`docs/recipe-prebaked-image-kit.md`](../docs/recipe-prebaked-image-kit.md)
for the general pattern this kit follows.
