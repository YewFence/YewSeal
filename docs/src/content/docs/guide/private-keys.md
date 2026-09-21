---
title: External private key sources
---

YewSeal is not responsible for remote storage, upload, download, or distribution of private keys. The Age identity bundle needed for decryption is supplied by developers, machines, or the deployment environment; see [Configuration - reading private keys](/guide/configuration#reading-private-keys) for the resolution order.

Personal private key paths are passed via `--key-file` or environment variables and never written into `.yewseal.toml`. Without an explicit identity, YewSeal finally tries `.age/keys.txt` under the current working directory; `init` can still generate the initial key there, but the path is never registered in project config.

Different developers, workstations, and production environments may use different keys. `[recipients.registry]` and the per-file authorization sets register public recipients only; environments are not required to share one private key. Give each environment only the identities it actually needs; never commit private keys to version control or print them into CI logs.

## Generating a recipient key

Fine-grained authorization — one key per alias, repository, or environment — starts with generating keypairs: `age-keygen` creates each pair, the public key goes into `[recipients.registry]`, and the private key travels to wherever its owner keeps it (the sections below cover storage and distribution). For teammates who generate their own keys, see [working with a team](/guide/working-with-a-team).

The reference script keeps the private key off the terminal: it pipes the bare `AGE-SECRET-KEY-1...` line straight into the clipboard — no `# public key:` comment — so the copy you handle is the one you paste into your password manager. The private key file exists only for the split second between `age-keygen` and the clipboard redirect, and a `trap` deletes it even when a step fails.

```sh
#!/bin/sh
set -eu
umask 077

alias=deploy   # the registry alias the public key will be registered under

keydir=$(mktemp -d "/tmp/yews-$alias.key.XXXXXX")
key="$keydir/identity"
trap 'rm -f "$key"; rmdir "$keydir"' EXIT
age-keygen -o "$key"
grep '^AGE-SECRET-KEY-' "$key" | wl-copy   # bare key line to clipboard; paste it into the password manager now
age-keygen -y "$key" > "/tmp/yews-$alias.pub"

echo "public key: /tmp/yews-$alias.pub"
```

`wl-copy` is Wayland-only; swap in your platform's clipboard CLI (`pbcopy` on macOS, `xclip -selection clipboard` on X11) — the platform matrix is the helper script's business. Printing the private key to stdout instead works over plain SSH but leaves it in terminal scrollback — prefer the clipboard. If the clipboard is overwritten before you save the key, just rerun the script: a fresh pair costs one command, and an unsaved private key has no recovery path by design. The repository ships the same flow as a helper script (`skills/yewseal/scripts/recipient-keygen.sh`) for agent-assisted setups.

## Infisical reference script

When private keys are hosted in Infisical, you can independently use the [Infisical CLI](https://infisical.com/docs/cli/commands/secrets) to export a secret's full value. Install the CLI and complete a login or machine identity first; authentication, access control, and secret contents are all managed by Infisical.

The POSIX shell script below is a reference to sync your age key via Infisical in different environments. Replace the project ID, environment, path, and secret name with the values for the current machine.

```sh
#!/bin/sh
set -eu
umask 077

mkdir -p .age
tmp=$(mktemp .age/keys.txt.XXXXXX)
trap 'rm -f "$tmp"' EXIT
trap 'exit 1' HUP INT TERM

infisical secrets get AGE_KEY_FILE --plain --silent \
  --projectId 'your-project-id' \
  --env 'dev' \
  --path '/yewseal' > "$tmp"
test -s "$tmp"
mv -f "$tmp" .age/keys.txt

yews --key-file .age/keys.txt decrypt
```

The script replaces the target only after a successful non-empty export, so a failing remote command never wipes an existing key; the temp file and the replaced file are mode `0600`. It does not validate the key format — YewSeal validates on read. Run it in a working directory you control and add `.age/` to `.gitignore`.

Configure `INFISICAL_TOKEN` or another authentication method per the [official Infisical docs](https://infisical.com/docs/cli/overview). For Docker, export the key on the host first, mount the file read-only into the container, and point `--key-file` at the in-container path.

## Exporting a local key

The other direction is just as common: after `yews init` generates a fresh identity on a new machine, back the key up to the channel you trust, or hand it to the deployment environment. Export the complete `.age/keys.txt` file — it may hold several identities — not a single line of it.

The reference script below mirrors the Infisical download above and pushes the local bundle into the same secret:

```sh
#!/bin/sh
set -eu

keyfile=$PWD/.age/keys.txt
test -s "$keyfile"

infisical secrets set "AGE_KEY_FILE=@$keyfile" \
  --projectId 'your-project-id' \
  --env 'dev' \
  --path '/yewseal' \
  --silent
```

`set` creates or overwrites the secret, and the `@file` form makes the CLI read the value from the file directly, so the key never appears in argv, shell history, or terminal output. Overwriting replaces the remote bundle wholesale: if the stored value held identities other than yours, merge them into the local file before pushing.

For a CI runner that consumes the key through an environment variable, the GitHub counterpart of the same export is one command — this stores the value the [CI/CD integration](/guide/ci-cd) page consumes as `AGE_KEY`:

```bash
gh secret set AGE_KEY < .age/keys.txt
```

## Per-identity secrets

The scripts above synchronize the whole bundle as one value, which couples every identity on the machine. The alternative is one secret per identity, named `YEWS_{alias}` after the registry alias: each environment pulls exactly the identities it needs, and each holder pushes only their own key — an overwrite can never clobber identities someone else pushed. The two reference scripts below implement that scheme; replace the project ID, environment, path, and secret names with your own.

The push script consumes `yews identities --json --reveal` instead of parsing key files or `.yewseal.toml` itself; chain resolution and the registry lookup are covered in [Configuration - reading private keys](/guide/configuration#reading-private-keys). The `# public key:` comment lines that `yews init` writes stay useful for humans reading `.age/keys.txt` — and the pull script recreates them — but nothing requires them anymore.

Each secret value stays a single bare `AGE-SECRET-KEY-1...` line. Comments never leave the local file, and Infisical's per-secret comment field is not used either: the CLI cannot set it, so anything stored there has to be maintained in the WebUI and no script can rely on it. Both scripts need only plain python3, and only to read `yews` output. Neither parses `.yewseal.toml`, so where the config lives is entirely YewSeal's business.

```sh
#!/bin/sh
set -eu
umask 077

tmp=$(mktemp)
report=$(mktemp)
pairs=$(mktemp)
trap 'rm -f "$tmp" "$report" "$pairs"' EXIT
trap 'exit 1' HUP INT TERM

yews identities --json --reveal > "$report"

python3 - "$report" > "$pairs" <<'PY'
import json, sys

with open(sys.argv[1]) as f:
    report = json.load(f)
for identity in report["identities"]:
    if identity.get("alias"):
        print(identity["alias"], identity["secret"])
PY

while read -r alias secret; do
  printf '%s\n' "$secret" > "$tmp"
  infisical secrets set "YEWS_$alias=@$tmp" \
    --projectId 'your-project-id' \
    --env 'dev' \
    --path '/yewseal' \
    --silent
  echo "pushed YEWS_$alias"
done < "$pairs"
```

The report flows through mode-`0600` temporary files, so secret keys never touch argv, shell history, or terminal output. An identity whose public key has no registry alias is warned on stderr by `yews identities` and skipped, so a borrowed or unregistered key is never uploaded under a wrong name. The script pushes whichever identities YewSeal would actually decrypt with — to push a different layer, point `--key-file` or `YEWSEAL_AGE_IDENTITIES` at it first.

The pull direction takes a comma-separated alias list and rebuilds `.age/keys.txt` — the path YewSeal reads by default, so no environment variable is needed afterwards. For each alias it fetches `YEWS_{alias}`, hands the value to `yews identities --key-file` and writes the public key that comes back as the `# public key:` comment, so the comment always describes the key right below it. The rebuilt file is exactly what the push script expects, so pull → push round-trips:

```sh
#!/bin/sh
set -eu
umask 077

: "${ALIASES:?set ALIASES to a comma-separated alias list, e.g. ALIASES=owner,ci $0}"

mkdir -p .age
one=$(mktemp)
report=$(mktemp)
bundle=$(mktemp .age/keys.txt.XXXXXX)
trap 'rm -f "$one" "$report" "$bundle"' EXIT
trap 'exit 1' HUP INT TERM

for alias in $(printf '%s\n' "$ALIASES" | tr ',' ' '); do
  infisical secrets get "YEWS_$alias" --plain --silent \
    --projectId 'your-project-id' \
    --env 'dev' \
    --path '/yewseal' > "$one"
  test -s "$one" || { echo "YEWS_$alias came back empty or missing" >&2; exit 1; }
  yews identities --key-file "$one" --json > "$report"
  public_key=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["identities"][0]["public_key"])' < "$report")
  printf '# public key: %s\n' "$public_key" >> "$bundle"
  cat "$one" >> "$bundle"
done

test -s "$bundle"
mv -f "$bundle" .age/keys.txt
```

The bundle file only appears once every fetch succeeded, every fetched value parsed as an age identity, and the result is non-empty; it is mode `0600`. YewSeal splits identities on commas, spaces, and newlines alike, so multi-line secret values or a hand-joined bundle all parse.

The `YEWS_{alias}` name is a naming convention the scripts trust rather than check: nothing verifies that the secret fetched under an alias really holds the identity registered for it. Audit the result with `yews identities`.

For a one-off shell or a CI runner, skip the file and join the per-identity values with commas straight into the environment variable — command substitution keeps the values out of shell history, and on GitHub Actions the repository secrets mirror the Infisical names upper-cased:

```bash
export YEWSEAL_AGE_IDENTITIES="$(infisical secrets get YEWS_owner --plain --silent --projectId your-project-id --env dev --path /yewseal),$(infisical secrets get YEWS_deploy --plain --silent --projectId your-project-id --env dev --path /yewseal)"
```

```yaml
env:
  YEWSEAL_AGE_IDENTITIES: ${{ secrets.YEWS_OWNER }},${{ secrets.YEWS_DEPLOY }}
```

## Other sources

Password managers, cloud secret managers, CI secrets, and local files all follow the same division of responsibility: the external tool provides the identity, YewSeal uses it to decrypt.
