# VCS-Assisted Config Discovery Degradation

## Status

Accepted. This decision defines the failure behavior for repository-aware configuration discovery and the boundary between configuration loading and `verify`'s version-control audit.

## Decision

Git and jj are optional infrastructure for repository-wide configuration discovery. They must improve discovery when available, but their command-line clients are not hard startup dependencies for ordinary YewSeal commands.

When repository-wide enumeration succeeds, YewSeal discovers all eligible configuration files in the repository and merges them in deterministic order. When repository detection or file enumeration is unavailable, YewSeal falls back to the existing direct search from the repository root to the current working directory. The command continues with that partial configuration set and emits an explicit warning on stderr.

A degraded discovery result is successful configuration loading, but it is not equivalent to complete project discovery. The loaded configuration carries both a machine-readable degraded state and human-readable warnings so callers cannot confuse the two states.

`verify` remains strict about the checks it reports, but it does not become unavailable. If configuration discovery was degraded, `verify` records a `vcs_query_failed` error finding for the version-control layer and continues its ciphertext, recipient, decryption, plaintext, and SOPS configuration checks. This produces a report and exit status 1 rather than turning the invocation into a configuration calling error with exit status 2.

## Rationale

Configuration discovery is a prerequisite for every business command. Making it depend unconditionally on an installed and compatible Git or jj CLI would turn an optional repository integration into a global availability requirement. That failure mode is disproportionate: a missing CLI should reduce repository-wide discovery, not make a directly usable configuration inaccessible.

The fallback preserves the pre-repository behavior and the existing root-to-current-directory lookup. It also keeps the limitation visible, so a command run from the repository root cannot silently claim that it processed every subdirectory configuration.

`verify` has a different responsibility. Its version-control layer exists specifically to audit tracked and pending sensitive files, so an unavailable VCS query must be reported as an error finding. Other verification layers remain useful and must still run.

## Observable behavior

- A missing Git or jj executable does not prevent `plan`, `encrypt`, `decrypt`, `clean`, `edit`, `view`, or `diff` from loading directly discoverable configuration.
- Repository-wide discovery warnings go to stderr; JSON command output on stdout remains machine-readable.
- If no directly discoverable configuration exists, the command still fails with the normal missing-configuration calling error.
- A configuration file loaded during degraded discovery is still parsed, merged, and validated normally.
- `verify` includes `vcs_query_failed` in its report when discovery is degraded and continues all independent checks.
- `--no-decrypt` and `--sync-sops-config=false` do not disable or hide the degraded VCS discovery finding.

## Non-goals

This decision does not require reimplementing Git or jj repository semantics in the configuration package. VCS adapters may continue to use their native clients for repository-wide enumeration. Configuration discovery and verification intentionally use separate repository queries: discovery reads only eligible current files, while verification additionally reads parent revisions to classify history. jj discovery uses a non-integrating snapshot with automatic tracking restricted to `.yewseal.toml`, so discovering a new config does not persist unrelated plaintext or key files into the working-copy commit. Replacing these adapters with library-backed implementations is a separate optimization and must preserve the complete-versus-degraded contract.

This decision does not widen command selection. Degraded discovery uses only the configurations that can be found through the direct root-to-current-directory search; it does not recursively scan the filesystem as an unannounced substitute.
