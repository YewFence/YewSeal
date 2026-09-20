---
title: yews clean
---

Safely remove registered local plaintext files

## Synopsis

Remove registered local plaintext, by default only after proving that its
ciphertext can be decrypted with the current Age identities. clean never
encrypts files, changes recipients, repairs project metadata, removes
directories, or displays plaintext or diff content.

Target selection uses the plaintext side for current-directory scope,
directories, and patterns. Exact registered plaintext or ciphertext paths both
select a mapping; multiple selectors take the union and every selector must
match. Dynamic groups discover the union of plaintext and ciphertext sides, so
a plaintext with missing ciphertext fails safely and a previously cleaned
ciphertext-only mapping is reported as already absent.

Matching decrypted bytes are removed automatically. Different bytes prompt
once per file with No as the safe default. --skip-different keeps every
difference without reading stdin; --remove-different removes differences
without reading stdin. Both --remove-different and an interactive Yes decrypt
the ciphertext again before removal and fail if it changed after the initial
comparison. These flags are mutually exclusive. Every existing plaintext must
be decrypted before removal and therefore requires a usable identity. Missing
or damaged ciphertext, no usable or matching identity, non-regular plaintext,
I/O errors, and a failed final recheck mark the item FAILED and retain it;
neither policy bypasses these checks.

--force is a separate, dangerous mode: it removes every selected plaintext
that currently exists without reading ciphertext, resolving identities,
comparing content, or prompting. It still follows the configured plaintext
path through symlinks, deletes only the same final regular file it inspected,
continues after per-file failures, and never removes directories. --force is
CLI-only, has no short form or environment variable, and is mutually exclusive
with --remove-different and --skip-different. Removed plaintext may not be
recoverable.

Plaintext paths may contain symlinks. clean follows the complete chain, removes
only the final regular-file target, and leaves links in place. Broken links are
already absent. Immediately before normal removal it resolves the chain and
reads the target again; a changed target, type, or byte snapshot is retained as
a failure. --force instead confirms that the chain still reaches the same
regular file without reading its content. Completed removals are not rolled
back when another item later fails.

Output: stdout is always empty. Prompts, warnings, REMOVED/RETAINED/FAILED
results, and the final summary go to stderr; --verbose also prints selection
details and ALREADY ABSENT results. clean does not update .gitignore or
.sops.yaml.

Exit codes: 0 when the selected policy completes, including explicitly retained
differences; 1 when any item, prompt, or output channel fails (earlier
removals remain); 2 for calling errors: invalid arguments, config, selection,
an unreadable explicit key file, or a failed key command. An empty identity
set instead makes each plaintext that needs verification fail safely with exit
1; no file is removed. Identity-source errors do not apply to --force because
that mode does not resolve identities.

See also: "yews diff" to inspect a difference before deciding and "yews
encrypt" to save local changes before cleaning.

Documentation: https://yewfence.github.io/YewSeal/guide/plaintext-cleanup

```
yews clean [command options] [path-or-pattern]... [flags]
```

## Examples

```
  # Clean mappings in the current directory scope
  yews clean

  # Clean one mapping selected by its plaintext path
  yews clean config.toml

  # Select registered plaintext paths with a pattern
  yews clean './configs/*.toml'

  # Keep all differences without prompting
  yews clean --skip-different

  # Irreversibly remove differences after successful decryption
  yews clean --remove-different

  # DANGEROUS: remove every selected plaintext without recovery checks
  yews clean --force

  # Inspect one difference before cleaning it
  yews diff -- config.toml
```

## Options

```
      --force              DANGEROUS: remove all selected plaintext without recoverability checks (CLI only)
  -h, --help               help for clean
      --remove-different   Remove plaintext that differs from successfully decrypted ciphertext without prompting (env YEWSEAL_CLEAN_REMOVE_DIFFERENT)
      --skip-different     Keep plaintext that differs from successfully decrypted ciphertext without prompting (env YEWSEAL_CLEAN_SKIP_DIFFERENT)
  -v, --verbose            Enable verbose output (selection info and already-absent results on stderr) (env YEWSEAL_CLEAN_VERBOSE)
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, YEWSEAL_AGE_KEY_CMD, SOPS_AGE_KEY_CMD, then .age/keys.txt in the current directory) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
