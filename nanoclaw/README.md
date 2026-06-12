# nanoclaw

A standalone sandbox kit (`kind: sandbox`, the v2 spec naming) for
[nanoclaw](https://github.com/nanocoai/nanoclaw) — a lightweight AI
assistant runtime that runs its agents in their own containers.

Unlike the previous version of this kit (which cloned and built nanoclaw at
sandbox creation, 5-12 minutes on first boot), this kit uses a **pre-baked
sandbox image**: the nanoclaw checkout, compiled build, Node 22, pnpm, the
OneCLI CLI, and the inner container images (nanoclaw-agent, OneCLI gateway,
postgres) all ship inside the image. The kit itself only applies policy —
network rules, credential proxying, published ports — so a new sandbox is
chatting in well under a minute.

## Usage

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=nanoclaw" nanoclaw
```

On attach the entrypoint seeds the inner Docker daemon from the embedded
image archive (first boot only, ~30s), starts the nanoclaw service, and
drops you into the setup wizard for OneCLI registration, auth, and channel
pairing. Chat platform adapters (WhatsApp, Telegram, Discord, Slack, …)
are installed via `/add-<channel>` skills from inside the session.

## Published ports

The kit declares these in-container ports for publishing to ephemeral
host ports:

| Port  | Name             | Purpose                                  |
|-------|------------------|------------------------------------------|
| 3000  | webhook          | Chat-platform webhook callbacks (Slack, Teams, GitHub, …) |
| 10254 | onecli-dashboard | OneCLI dashboard / API                   |
| 10255 | onecli-gateway   | OneCLI credential gateway                |

> `sbx` v0.32.0 validates `publishedPorts` but does not yet bind them
> automatically at sandbox start. Until that lands, publish manually:
>
> ```console
> $ sbx ports <sandbox> --publish 3000/tcp --publish 10254/tcp --publish 10255/tcp
> $ sbx ports <sandbox>   # shows the assigned host ports
> ```

## How auth works

The kit uses the standard Anthropic auth wiring: `serviceDomains`/`serviceAuth`
for `api.anthropic.com`, the OAuth flow against `platform.claude.com`, and the
`proxy-managed` sentinel pattern. Credentials never enter the container — the
sandbox proxy substitutes the real value on egress.

Set `ANTHROPIC_API_KEY` in your environment to skip OAuth and use an API key
directly.

## Image architecture

```
docker.io/ealeyner/nanoclaw-sbx
└── FROM docker/sandbox-templates:claude-code-docker
    ├── Node 22 + pnpm 10
    ├── /home/agent/nanoclaw      checkout @ pinned ref, pnpm install + tsc build
    ├── ~/.local/bin/onecli       OneCLI CLI binary
    └── /opt/nanoclaw/images.tar  inner images, docker-loaded at first boot:
        ├── nanoclaw-agent:latest          (built from nanoclaw/container)
        ├── ghcr.io/onecli/onecli:<pinned> (credential gateway)
        └── postgres:18-alpine             (gateway database)
```

The `CONTAINER_IMAGE=nanoclaw-agent:latest` environment override makes
nanoclaw use the pre-loaded agent image instead of building its own
per-checkout tag, and `NANOCLAW_SKIP=service,container` keeps the setup
wizard from redoing pre-baked steps.

## Building and publishing the image

```console
$ ./scripts/build-image.sh
```

This clones nanoclaw at the pinned ref (`NANOCLAW_REF`), builds the agent
container image, pulls the OneCLI gateway images, saves all three into
`images/<arch>/images.tar`, and builds the sandbox image. Override the tag
with `IMAGE=docker.io/<you>/nanoclaw-sbx:latest`, then `docker push`.

To bump the baked nanoclaw version: update `NANOCLAW_REF` in
`scripts/build-image.sh` (and `ONECLI_VERSION` if nanoclaw's
`setup/onecli.ts` pin moved), rebuild, push, and update `version:` in
`spec.yaml`.

## Testing locally without pushing

```console
$ IMAGE=docker.io/ealeyner/nanoclaw-sbx:latest ./scripts/build-image.sh
$ docker save docker.io/ealeyner/nanoclaw-sbx:latest -o /tmp/nanoclaw-sbx.tar
$ sbx template load /tmp/nanoclaw-sbx.tar
$ sbx run --kit . nanoclaw
```

## Debugging

```console
$ sbx exec <sandbox> -- tail -f /home/agent/nanoclaw/logs/nanoclaw.error.log
$ sbx exec <sandbox> -- cat /tmp/nanoclaw-load-images.log   # first-boot image seeding
$ sbx exec <sandbox> -- docker images                       # inner images present?
```
