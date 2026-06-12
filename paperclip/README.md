# paperclip

A standalone sandbox kit (`kind: sandbox`, the v2 spec naming) for
[Paperclip](https://github.com/paperclipai/paperclip) — the open-source
app for managing AI agents at work: a Node.js server + React UI that
orchestrates a team of agents ("if OpenClaw is an employee, Paperclip is
the company").

The kit uses a **pre-baked sandbox image**: Node 22 and the pinned
`paperclipai` package (server, built UI, embedded PostgreSQL binaries)
ship inside the image, built on the `claude-code` template so the
`claude_local` adapter has Claude Code available out of the box. A new
sandbox serves the web UI in seconds.

## Usage

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=paperclip" paperclip
$ sbx ports <sandbox> --publish 3100/tcp   # then open the printed host port
```

On attach the entrypoint runs `paperclipai onboard --yes` — idempotent:
first boot writes config (instance, agent JWT secret, secrets key) under
`~/.paperclip` and starts the server; later boots just start the server.
The kit runs in **authenticated mode** (upstream's Docker default) with a
generated, persisted `BETTER_AUTH_SECRET` — create your account on first
UI visit. (Paperclip's zero-auth `local_trusted` mode hard-requires a
loopback bind, which the sandbox port-forwarder can't reach.)

## Published ports

| Port | Name | Purpose |
|------|------|---------|
| 3100 | web  | REST API + web UI + WebSocket (single port) |

Embedded PostgreSQL stays on loopback :54329 inside the sandbox.

> `sbx` v0.32.0 validates `publishedPorts` but does not yet bind them
> automatically — publish manually as shown above.

## How auth works

Agent adapters spawn provider CLIs in-container; the Anthropic wiring
(`serviceDomains`/`serviceAuth`/`proxyManaged`) lets the sandbox proxy
inject `ANTHROPIC_API_KEY` on egress for the `claude_local` adapter.
Other provider keys (OpenAI, Gemini, …) can be added as sandbox secrets
or configured in the UI.

Telemetry is opted out at the source (`PAPERCLIP_TELEMETRY_DISABLED=1`);
`telemetry.paperclip.ing` is deliberately not in `allowedDomains`.

## Image architecture

```
docker.io/ealeyner/paperclip-sbx
└── FROM docker/sandbox-templates:claude-code
    ├── Node 22 (paperclip requires >= 20)
    └── paperclipai @ pinned version   npm global install:
        ├── @paperclipai/server + built React UI
        ├── embedded-postgres (chowned to agent — it creates lib
        │   symlinks in its package dir on first run)
        └── /usr/local/bin/paperclipai symlink
```

## Building and publishing the image

```console
$ PAPERCLIP_VERSION=2026.609.0 IMAGE=docker.io/<you>/paperclip-sbx:latest ./scripts/build-image.sh
$ docker push docker.io/<you>/paperclip-sbx:latest
```

## Debugging

```console
$ sbx exec <sandbox> -- tail -f /home/agent/.paperclip/instances/default/logs/*.log
$ sbx exec <sandbox> -- curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3100/
$ sbx exec <sandbox> -- paperclipai doctor
```

See [`docs/recipe-prebaked-image-kit.md`](../docs/recipe-prebaked-image-kit.md)
for the general pattern this kit follows.
