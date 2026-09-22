---
title: Agent skills
---

YewSeal ships two agent skills in the `skills/` directory of the repository. Installed into a project's `.agents/skills/`, they teach a coding agent the YewSeal workflow: what it may do on its own (read the config, register recipients, encrypt, validate) and what always requires your explicit go-ahead (anything that touches plaintext).

This step is optional. The CLI works exactly the same without the skills; they only make agent collaboration safer and better informed.

## What gets installed

- `yewseal` — the infrastructure side. The agent may invoke it on its own whenever it works in a YewSeal repository: selection rules, recipient registration, encryption, validation, and the documentation paths.
- `yewseal-secrets` — the plaintext side. It carries `disable-model-invocation: true`, so no harness ever offers it to the agent automatically: it loads only when you explicitly ask for `yewseal-secrets` by name, and that explicit call is what authorizes plaintext work. An agent merely reading the file — during installation, a directory scan, anything else — grants no authorization; the skill text states this rule itself.

## Installing

The repository ships an installer, `scripts/install-agent-skills.sh`. It reads the version from your installed `yews` binary, then fetches only the files under `skills/` from that release tag via the GitHub API (no full-repository download) into `./.agents/skills/` of the current directory. Skills and binary therefore always match: the installer never falls back to `main`.

Fetch the script, review it — it is short — then run it:

```bash
gh api repos/YewFence/YewSeal/contents/scripts/install-agent-skills.sh \
  -H "Accept: application/vnd.github.raw" > install-agent-skills.sh
less install-agent-skills.sh
bash install-agent-skills.sh
```

It requires the GitHub CLI (`gh`) to be authenticated. Run it again after upgrading the CLI to refresh the skills. Commit `.agents/skills/` if you want the whole team on the same guidance — teammates still need their own private keys, and `yewseal-secrets` still loads only on your explicit request.

## Letting your agent install them

If you would rather delegate, paste this to your coding agent:

> Fetch `scripts/install-agent-skills.sh` from the YewFence/YewSeal repository on GitHub, show me what it does, and once I confirm, run it to install the YewSeal agent skills into this project's `.agents/skills/`.

## Why the skills are not embedded in the CLI

The skills deliberately live only in the repository — no `yews` subcommand prints or installs them. CLI output is a channel an agent can reach on its own, and `yewseal-secrets` must stay reachable only through you invoking it by name; keeping it out of the binary leaves no side door around that authorization boundary.
