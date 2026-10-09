---
title: Delivering plaintext to runtime consumers
---

`decrypt` writes files and stops there. Starting consumers, injecting their environment, and managing the output directory's lifetime are the caller's job. The scripts below are reference compositions you can adapt: the caller creates the output root, launches consumers only after decryption succeeds, and cleans up on both success and failure. Plaintext lifetime outside the registered project paths stays with the caller.

For command options and exit codes, see [`yews decrypt --help`](/references/yews_decrypt). For selecting files, use the shared [target-selection rules](/guide/target-selection).

## The mirrored output tree

`decrypt --output DIR` delivers the selected mappings under another root. Selectors choose **what**; the output root chooses **where**. Every destination is `DIR / relative(project-root, configured-plaintext-path)`. The project root is the repository root used during config discovery, or the current config discovery directory outside a repository. There is no repository-name wrapper, flattening, or renaming: `.env` goes to `DIR/.env`, and `compose/app.yaml` goes to `DIR/compose/app.yaml`. A single selected file keeps the same project-relative path, whether selected from the root or a subdirectory and regardless of which config declared it.

Selected logical plaintext paths outside that project root and conflicting destinations reject the whole command before any plaintext is written. Every destination is a regular file at the logical registered path; source symlink structure is not reproduced. Formats remain those of the registered mappings.

The caller supplies the output root: a real, empty directory. A missing root, a file, a symlink, or any existing entry — including a hidden one — is rejected. YewSeal creates new child directories with `0700` and plaintext files with `0600`; the root itself stays as the caller created it, and YewSeal never removes it. Use a trusted, one-use directory.

`--output` is mutually exclusive with `--force=true` and `--inplace=true`. `YEWSEAL_DECRYPT_OUTPUT` supplies the directory, resolved relative to the calling working directory; an explicit flag wins. Parallel workers, verbose diagnostics, identity sources, and strict mode work as usual.

:::caution
An output root inside the repository is allowed. Delivery writes the tree without consulting Git ignore rules, warning about the placement, or updating `.gitignore` or `.sops.yaml`, so preventing accidental commits is the caller's responsibility. Configured-path decryption retains its existing ignore behavior.
:::

## A complete runtime snapshot with Shell cleanup

The default is [lenient decryption](/guide/decryption-results): unavailable or non-matching identities may leave a partial tree even on exit `0`. For runtime delivery, always use `--strict` and launch the consumer **only after decrypt succeeds**. Successful writes survive strict failure and real errors, so cleanup must cover failure as well as success. [`clean`](/guide/plaintext-cleanup) handles registered plaintext locations only; delivery trees are outside its scope.

This POSIX-shell example uses `/dev/shm` on a Linux host where it is mounted as tmpfs. Pick another trusted absolute temporary base for your environment; a regular disk-backed directory also works. When you change the base, update the `case` pattern in `cleanup` to match it — the guard refuses to clean anything outside the expected base.

```sh
#!/bin/sh
set -eu
umask 077

YEWS_TMP=$(mktemp -d /dev/shm/yews.delivery.XXXXXX)
cleanup() {
    if [ -n "${YEWS_TMP:-}" ]; then
        case "$YEWS_TMP" in
            /dev/shm/yews.delivery.?*) rm -rf -- "$YEWS_TMP" ;;
            *) printf '%s\n' 'Refusing to clean an unexpected delivery path' >&2; return 1 ;;
        esac
    fi
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
export YEWS_TMP

yews decrypt --strict --output "$YEWS_TMP"
docker compose --project-directory "$PWD" \
    --env-file "$YEWS_TMP/.env" \
    -f "$YEWS_TMP/compose/app.yaml" up --abort-on-container-exit
```

Here `.env` and `compose/app.yaml` are registered plaintext paths, and the command runs from the project root. Replace Compose with your own consumer as needed. Compose resolves relative paths using the chosen project directory; for bind-mounted secrets, your Compose file can refer to `${YEWS_TMP}/secrets/prod.env`. The caller exports that variable, not YewSeal. Keep the tree alive until every consumer has finished reading it; avoid detached consumers that still need the files after the script exits.

The traps cover normal exit and catchable termination signals. Cleanup errors still need attention from the caller. Never enable Shell tracing around secrets or echo decrypted content into CI logs.

## Delivery-classified files

Declare runtime-sensitive mappings with [`plaintext_mode = "delivery"`](/guide/configuration#plaintext-classification) so default decrypt routes them to `--output` instead of their configured plaintext path. The configured `secrets/prod.env`, for example, can be delivered to the tmpfs tree above without appearing at its ordinary plaintext location. `plan` and `verify` expose `plaintext_mode` for audits.

:::caution
Delivery classification is an accident-prevention default. Anyone with a matching identity can still use `view` or add `--inplace`; use separate recipients and identities for access control.
:::

For a mapping containing a raw token with no trailing newline, consume stdout directly and export only after successful decryption:

```sh
TOKEN=$(yews view secrets/token.enc.bin)
export TOKEN
```

Use this in a script with `set -e`, or check the assignment's status before proceeding. Shell command substitution strips trailing newlines and carries arbitrary binary data unreliably; environment variables stay accessible to the process and its children. Export one token per assignment, not a whole ENV-format file.

If a tool requires the configured project location, explicitly decrypt it there and close the session with safe cleanup:

```sh
yews decrypt secrets/prod.enc.env --inplace
yews clean secrets/prod.enc.env
```

Run your consumer between these commands, and arrange cleanup for failures too. Saving local edits back to ciphertext is a separate `encrypt` step; `clean` protects differences by default.

## Arbitrary filenames with `view`

`view` accepts exactly one registered mapping and emits plaintext to stdout; it updates no metadata and has no file or directory output. A Shell redirect such as `yews view config.enc.toml > final-file` loses YewSeal's file-writing protection: the Shell may open and truncate an existing file before decryption even begins, new-file permissions depend on its `umask`, and a failed decryption can leave an empty target.

Instead, create a unique, permission-controlled temporary file beside the intended final target, and publish it only after `view` succeeds. This example deliberately replaces the final target after success; choose a publication policy appropriate to your application.

```sh
#!/bin/sh
set -eu
umask 077

final=./runtime-config.toml
staged=$(mktemp "${final}.tmp.XXXXXX")
cleanup() {
    if [ -n "${staged:-}" ]; then
        rm -f -- "$staged"
    fi
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

yews view config.enc.toml > "$staged"
mv -f -- "$staged" "$final"
staged=
```

`mktemp` creates the temporary file with restrictive permissions; the failed decryption never opens the final target. The script owns replacement, staging cleanup, and the final file's later lifetime. If the existing final target is a directory or an untrusted symlink, extend this pattern to handle those cases explicitly.
