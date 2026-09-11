---
layout: home

hero:
  name: YewSeal
  text: Encrypted configuration file manager
  tagline: Encrypt the TOML, YAML, JSON, ENV, and INI configuration files in your repository with SOPS and Age, so sensitive settings can be committed safely.
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: CLI reference
      link: /references/yews

features:
  - title: Multi-format configuration
    details: Native handling of TOML, YAML, JSON, ENV, and INI with no format conversion or external tools.
  - title: Embedded crypto
    details: Ships the SOPS encryption engine and Age key generation, minimizing external tool dependencies.
  - title: Config-driven batches
    details: Declare exact file mappings and group scans in .yewseal.toml, and preview the selection with plan.
  - title: Per-file authorization
    details: Configure each file's recipient set from public keys; decryption identities are provided by developers or deployment environments.
---
