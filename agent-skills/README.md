# agent-skills

A mixin kit (extends `claude`) installing
[addyosmani/agent-skills](https://github.com/addyosmani/agent-skills) —
24 production-grade engineering skills for AI coding agents, spanning
Define, Plan, Build, Verify, Review, and Ship.

## Usage

```console
$ sbx run --kit "git+https://github.com/docker/sbx-kits-contrib.git#dir=agent-skills" claude
```

Skills land in `~/.claude/skills/` (pinned to upstream tag `0.6.2`) and
Claude Code discovers them automatically.

## Design notes

This is a **skills-only copy** — the kit deliberately skips upstream's
plugin install path (`.claude-plugin/`, slash commands, hooks):

- the plugin's `/review`, `/ship`, `/spec`, `/test`, `/plan` commands
  would collide with other skill packs (e.g. the gstack kit);
- the SessionStart hook needs `jq` and adds little in a sandbox;
- `SKILL.md` files are self-contained (only `idea-refine/` carries a
  helper script, which is offline and dependency-free).

Composes cleanly with other skill packs — the long hyphenated skill
names don't collide.

No telemetry; only github.com is contacted, at install time.
