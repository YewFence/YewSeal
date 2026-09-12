---
title: YewSeal
description: Encrypted configuration file manager
template: splash
hero:
  tagline: Encrypt the TOML, YAML, JSON, ENV, and INI configuration files in your repository with SOPS and Age, so sensitive settings can be committed safely.
  actions:
    - text: Get started
      link: guide/getting-started/
      variant: primary
    - text: CLI reference
      link: references/yews/
      variant: secondary
---

YewSeal is a configuration file encryption manager built on SOPS and Age, supporting TOML, YAML, JSON, ENV, INI, and binary files. Every format (TOML included) is encrypted natively by the embedded SOPS engine with no format conversion.

## Multi-format configuration

Native handling of TOML, YAML, JSON, ENV, and INI with no format conversion or external tools.

## Embedded crypto

Ships the SOPS encryption engine and Age key generation, minimizing external tool dependencies.

## Config-driven batches

Declare exact file mappings and group scans in .yewseal.toml, and preview the selection with plan.

## Per-file authorization

Configure each file's recipient set from public keys; decryption identities are provided by developers or deployment environments.
