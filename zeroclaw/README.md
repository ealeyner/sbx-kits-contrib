# zeroclaw

A standalone sandbox kit (`kind: sandbox`, the v2 spec naming) for
[ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) — fast, small,
fully autonomous AI assistant infrastructure in Rust: a single binary
running a gateway with 30+ channels and ~20 providers.

ZeroClaw ships pinned per-arch release binaries, so this kit deliberately
uses **no custom image**: one install command downloads the pinned
upstream release onto the stock `shell` template in seconds. (See
[`docs/recipe-prebaked-image-kit.md`](../docs/recipe-prebaked-image-kit.md)
for when a pre-baked image *is* worth it.)

## Usage

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=zeroclaw" zeroclaw
```

On attach the entrypoint runs `zeroclaw daemon` (gateway, channels,
scheduler, heartbeat). Talk to it over the gateway's WebSocket chat
(`/ws/chat` on :42617) or wire up channels in `~/.zeroclaw/config.toml`.

## Published ports

| Port  | Name    | Purpose |
|-------|---------|---------|
| 42617 | gateway | HTTP/WS gateway: `/health`, `/metrics`, `/ws/chat`, webhooks |

> The web dashboard isn't bundled in upstream's release binaries (only in
> their container image), so `/` returns 503 — the API and WS endpoints
> are fully functional. `sbx` v0.32.0 doesn't auto-bind `publishedPorts`
> yet; publish manually with `sbx ports <sandbox> --publish 42617/tcp`.

## How auth works

ZeroClaw v0.8.0 removed legacy `ANTHROPIC_API_KEY` env fallbacks; keys
live in `config.toml` (or the `ZEROCLAW_<dotted__path>` env grammar). The
kit seeds a config with a `__ANTHROPIC_API_KEY__` placeholder and
substitutes the proxy-managed sentinel at sandbox start — the sandbox
proxy injects the real credential on egress, so the secret never enters
the sandbox.

`sandbox_backend = "none"` is set because tool calls already run inside
the sandbox microVM (ZeroClaw's own Landlock/Bubblewrap backends aren't
available in-container).

## Debugging

```console
$ sbx exec <sandbox> -- curl -s http://127.0.0.1:42617/health
$ sbx exec <sandbox> -- zeroclaw status
```
