---
title: Glossary
---

Short definitions for looking up unfamiliar terms. Each entry can be read on its own; the links offer optional detail.

## Alias

A name in the [registry](#registry) for one public Age key, such as `owner` or `teammate`. See [recipient authorization](/guide/configuration#recipient-authorization) for accepted names and authorization lists.

## Discovery root

The directory of the config file that owns a group, also used to resolve that config's relative file paths. See [Configuration](/guide/configuration#group-scanning).

## Group

An `[[encryption.groups]]` entry that discovers multiple file mappings using gitignore-style patterns relative to its config directory. See [Configuration - group scanning](/guide/configuration#group-scanning).

## Identity

A private Age key that provides decryption access. Several identities can form a bundle. See [Configuration - reading private keys](/guide/configuration#reading-private-keys) for sources and their precedence, and [`yews identities`](/references/yews_identities) to inspect them.

## Lenient and strict

The two postures of `decrypt`. Lenient (the default) treats [skips](#skip) as acceptable, so people holding different [identities](#identity) can share one repository. Strict turns any skip into a failure, for deployment gates, and applies to the selected mappings — never the whole repository — so no identity needs to be a recipient of everything. `diff` has no strict mode — it is a development preview, not a gate. See [Decryption results and strict mode](/guide/decryption-results).

## Mapping

An `[[encryption.files]]` entry pairing one plaintext path with one encrypted path, optionally with `format` and `recipients`. Also called a *file pair*. See [Configuration - file mappings](/guide/configuration#file-mappings).

## Provenance

Where a resolved value came from. `plan --source` reports each path, format, authorization, and registry origin (e.g. `.yewseal.toml format`) in a describe block, so drift between config layers is visible before anything is written.

## Protocol file

An encrypted file with a format-specific `.enc.*` suffix, such as `config.enc.toml`. See [file naming conventions](/guide/configuration#file-naming-conventions) for the supported suffixes.

## Recipient

A public Age key a file is encrypted to. See [recipient authorization](/guide/configuration#recipient-authorization) for how the configured set is chosen, and [Working with a team](/guide/working-with-a-team) for changing access.

## Registry

`[recipients.registry]`: the committed map of [aliases](#alias) to public keys — the only source YewSeal accepts for encryption [recipients](#recipient). Private keys can never live here.

## Scope

The directory range a command considers. With no target arguments, this is the current working directory and its subdirectories. See [default selection](/guide/target-selection#default-selection) for command-specific rules.

## Selection

The set of mappings a command will process, produced by interpreting CLI arguments as selectors over what `.yewseal.toml` registered. Arguments only ever narrow; they never register files or declare authorization. See [Target selection](/guide/target-selection).

## Sides

The plaintext and encrypted paths of a mapping, such as `config.toml` and `config.enc.toml`. See [Target selection](/guide/target-selection#the-two-sides-of-a-mapping) for the side each command selects by.

## Skip

A per-file outcome meaning "no matching [identity](#identity)" for `decrypt`, or a missing input for `diff` — not an error in [lenient](#lenient-and-strict) mode, never a sign of corruption. See [Decryption results and strict mode](/guide/decryption-results).

`verify` reports skipped checks too, but that is a different concept: a layer that did not run — no identity available, outside a repository — not a per-file outcome. See [Verifying repository health](/guide/verifying).
