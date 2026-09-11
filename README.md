# YewSeal

YewSeal is a CLI built around **SOPS + Age** that defines batch encryption and decryption workflows through one clear config file.

It does not reinvent crypto. It wires project initialization, file-mapping registration, and batch encrypt/decrypt into a friendly CLI. Remote storage and distribution of private keys stay with developers and deployment environments.

## Features

- Built on SOPS + Age, compatible with native `.sops.yaml` and Age key workflows
- One intuitive TOML config declaring every `plaintext` / `encrypted` mapping in the project
- `init` scaffolds keys, config, and the recommended project files
- Short `yews e` / `yews d` aliases, config-wide batches, directory scans, and parallel workers
- TOML, YAML, JSON, ENV, INI, and binary; TOML is encrypted natively by the built-in TOML store (via the [YewFence/sops](https://github.com/YewFence/sops) fork)
- Decryption identities via flags, environment variables, or the default key file — private key locations never live in project config
- Original data structures survive encryption/decryption

## Installation

```bash
# mise (recommended)
mise use --global github:YewFence/YewSeal

# or go install
go install github.com/YewFence/YewSeal/cmd/yews@latest
```

Prebuilt binaries are on the [releases page](https://github.com/YewFence/YewSeal/releases); the executable is always named `yews`. To build from source, clone the repo and run `mise run install`. A Docker image is available too:

```bash
docker run --rm -it --user "$(id -u):$(id -g)" -v "$PWD:/work" \
  ghcr.io/yewfence/yew-seal:latest --help
```

## Quick start

```bash
# interactive init: age keys, .yewseal.toml, file registration
yews init

# encrypt every registered file
yews e

# decrypt from the same config
yews d
```

Commit the encrypted files to version control and back up `.age/keys.txt` — it holds the Age key pair, and losing it means losing the plaintext.

## Documentation

Every command is self-documenting: `yews <command> --help` carries the complete semantics (target selection, flags, exit codes, examples). The documentation site mirrors the same content plus guides:

- [Getting started](https://yewfence.github.io/YewSeal/guide/getting-started) — install, init, first encrypt/decrypt
- [Configuration](https://yewfence.github.io/YewSeal/guide/configuration) — `.yewseal.toml`, `.sops.yaml`, environment variables
- [Workflows](https://yewfence.github.io/YewSeal/guide/workflows) — how the commands compose in daily use
- [CLI reference](https://yewfence.github.io/YewSeal/references/yews) — generated from the same source as `--help`

For simple single-file tasks without a project config, use the standalone SOPS CLI; see [Interop with SOPS](https://yewfence.github.io/YewSeal/guide/sops).

## How it works

All formats (TOML included): plaintext file → sops encrypt/decrypt → encrypted file, through embedded libraries:

- **Age** ([`filippo.io/age`](https://github.com/FiloSottile/age)) — key generation
- **SOPS fork** ([`github.com/YewFence/sops/v3`](https://github.com/YewFence/sops)) — the encryption engine; the fork adds a native TOML store on top of upstream [getsops/sops](https://github.com/getsops/sops). Engine-level issues belong in the fork's tracker.

## Why YewSeal

I wrote this because I really did not want `wrangler.toml` sitting in a public GitHub repo — nothing fatal inside, but domain names, project IDs, KV/D1/R2 identifiers and the like felt wrong to expose. Infisical alone was inconvenient since every config tweak meant a trip to the website. SOPS looked right but has no native TOML support, and I refuse to live in YAML's indentation hell, so this orchestrator was born. Then came short aliases and persistent config, then every SOPS-supported format "while we're at it", then multi-file batches, then parallel workers because Go makes it cheap… and it grew into what it is now. Age keys round it out nicely: public keys live in the config, private keys stay local, and a secret manager covers CI/CD.

## Disclaimer

This is a personal learning project. It is neither thoroughly tested nor audited; do not use it where high-security guarantees are required.

## Contributing

Issues and pull requests are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License. Core functionality depends on these excellent open-source tools:

- [Age](https://github.com/FiloSottile/age) (BSD 3-Clause)
- [SOPS](https://github.com/getsops/sops) (MPL 2.0) — via [YewFence/sops](https://github.com/YewFence/sops), a fork with a native TOML store
