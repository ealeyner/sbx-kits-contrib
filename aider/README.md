# aider

A standalone sandbox kit (`kind: sandbox`, the v2 spec naming) for
[Aider](https://github.com/Aider-AI/aider) — AI pair programming in your
terminal. Aider runs inside your workspace git repo and auto-commits its
edits.

Aider installs in under a minute via `uv` (prebuilt wheels, parallel
downloads), so this kit deliberately uses **no custom image** — one
install command on the stock `shell` template. (See
[`docs/recipe-prebaked-image-kit.md`](../docs/recipe-prebaked-image-kit.md)
for when a pre-baked image *is* worth it.)

## Usage

From a git repo you want to pair on:

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=aider" aider .
```

On attach you land in the aider TUI with your workspace mounted. Aider
picks its default model from the available provider credentials.

## How auth works

The kit wires both Anthropic (`x-api-key`) and OpenAI (`Bearer`) through
the sandbox credential proxy. litellm requires the key env vars to
exist, so `proxyManaged` seeds sentinel values; the proxy substitutes
real credentials on egress — secrets never enter the sandbox. Set either
secret on the host:

```console
$ printf '%s\n' "$ANTHROPIC_API_KEY" | sbx secret set -g anthropic
$ printf '%s\n' "$OPENAI_API_KEY" | sbx secret set -g openai
```

Telemetry and update checks are disabled at the source
(`AIDER_ANALYTICS=false`, `AIDER_ANALYTICS_DISABLE=true`,
`AIDER_CHECK_UPDATE=false`); no analytics domains are allow-listed.

## Notes

- Pinned to `aider-chat==0.86.2` (the last PyPI release; bump the pin in
  `spec.yaml` deliberately).
- The `[browser]`/`[help]` extras (Streamlit GUI, torch-based help index)
  and Playwright web scraping are not installed — add them in-session
  with `uv tool install --python 3.12 "aider-chat[browser]==0.86.2"` plus
  the needed network allowances if you want them.

## Debugging

```console
$ sbx exec <sandbox> -- /home/agent/.local/bin/aider --version
$ sbx exec <sandbox> -- env | grep AIDER
```
