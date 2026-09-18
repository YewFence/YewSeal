---
title: Glossary
---

Short definitions of the terms the guides use with precision. Each entry links to the page that covers the topic in depth.

## Alias

A reviewable name in the [registry](#registry) standing for one public Age key, e.g. `owner` or `teammate`. Authorization lists (`recipients`) accept aliases only, never raw keys. Aliases are case-sensitive and must start with an ASCII letter.

## Discovery root

The directory of the config file that owns a [group](#group) — and the base against which every explicit file pair path in that config resolves. Group scans always run under their own discovery root; CLI arguments can never turn an arbitrary directory into a new one. See [Target selection](/guide/target-selection).

## Group

An `[[encryption.groups]]` entry that discovers many files at once by gitignore-style `patterns` under its [discovery root](#discovery-root), instead of listing [mappings](#mapping) one by one. Encrypted paths follow the [protocol file](#protocol-file) naming. See [Configuration - group scanning](/guide/configuration#group-scanning).

## Identity

A private Age key (or a bundle of several in one file) that can decrypt files. Identities come from `--key-file`, environment variables, or `.age/keys.txt` — never from the project config. See [Configuration - reading private keys](/guide/configuration#reading-private-keys).

## Lenient and strict

The two postures of `decrypt`. Lenient (the default) treats [skips](#skip) as acceptable, so people holding different [identities](#identity) can share one repository. Strict turns any skip into a failure, for deployment gates, and applies to the selected mappings — never the whole repository — so no identity needs to be a recipient of everything. `diff` has no strict mode — it is a development preview, not a gate. See [Decryption results and strict mode](/guide/decryption-results).

## Mapping

An `[[encryption.files]]` entry pairing one plaintext path with one encrypted path, optionally with `format` and `recipients`. Also called a *file pair*. Explicit mappings are exact: both paths come from the config, and the files do not need to exist on disk to be selectable by `plan` and `diff`.

## Provenance

Where a resolved value came from. The `plan` table reports a source column for the mapping, the format, and the authorization (e.g. `.yewseal.toml format`), so drift between config layers is visible before anything is written.

## Protocol file

An encrypted file named by the `.enc.*` protocol. Specifically, they are `.enc.toml`, `.enc.yaml`, `.enc.json`, `.enc.env`, `.enc.ini`, and `.enc.bin`. Group discovery always excludes protocol files from the plaintext side, so `encrypt` never double-encrypts.

## Recipient

A public Age key a file is encrypted *to*. The effective set resolves as file pair over group over `recipients.defaults`, each level fully replacing the previous one. Adding or removing a recipient is a config edit followed by `encrypt`; see [Working with a team](/guide/working-with-a-team).

## Registry

`[recipients.registry]`: the committed map of [aliases](#alias) to public keys — the only source YewSeal accepts for encryption [recipients](#recipient). Private keys can never live here.

## Scope

The directory-restricted view of registered [mappings](#mapping) a command operates in when invoked without arguments: the current directory and below, never the whole repository. See [Target selection](/guide/target-selection).

## Selection

The set of mappings a command will process, produced by interpreting CLI arguments as selectors over what `.yewseal.toml` registered. Arguments only ever narrow; they never register files or declare authorization. See [Target selection](/guide/target-selection).

## Sides

The two paths of a [mapping](#mapping): the *plaintext side* (`config.toml`) and the *encrypted side* (`config.enc.toml`). Commands differ in which side they look at — `encrypt` and `diff` match by plaintext, `decrypt` by ciphertext, `plan` by either. See [Target selection](/guide/target-selection).

## Skip

A per-file outcome meaning "no matching [identity](#identity)" for `decrypt`, or a missing input for `diff` — not an error in [lenient](#lenient-and-strict) mode, never a sign of corruption. See [Decryption results and strict mode](/guide/decryption-results).
