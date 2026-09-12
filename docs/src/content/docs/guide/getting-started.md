---
title: Getting started
---

YewSeal is a configuration file encryption manager built on SOPS and Age, supporting TOML, YAML, JSON, ENV, INI, and binary files. Every format (TOML included) is encrypted natively by the embedded SOPS engine with no format conversion.

## Installation

### mise (recommended)

Install the prebuilt release binary via the [github backend](https://mise.jdx.dev/dev-tools/backends/github.html) of [mise](https://mise.jdx.dev/):

```bash
mise use --global github:YewFence/YewSeal
yews --version
```

You can also declare `github:YewFence/YewSeal` in a project's `mise.toml` so teammates and CI share one pinned version.

### go install

```bash
go install github.com/YewFence/YewSeal/cmd/yews@latest
```

Requires `$GOPATH/bin` on your `$PATH`.

### GitHub release

Download the prebuilt binary for your system from the [releases page](https://github.com/YewFence/YewSeal/releases). The executable is always named `yews` (`yews.exe` on Windows).

### Build from source

```bash
git clone https://github.com/YewFence/YewSeal.git
cd YewSeal
mise run install    # installs into $GOPATH/bin
# or build only: mise run build → build/yews
```

### Docker

Without installing a binary, run the container image `ghcr.io/yewfence/yew-seal` directly; see [Running with Docker](/guide/docker).

## Initialize a project

Run `init` inside the project directory to set up the Age keys, `.yewseal.toml`, and the optional `.sops.yaml`:

```bash
yews init
```

The command generates `.age/keys.txt`, writes the Age public key into `.yewseal.toml`, interactively records one or more `[[encryption.files]]` mappings, and adds the private key directory and plaintext files to `.gitignore`.

### Non-interactive mode

For scripts, pass the plaintext and encrypted files of the first config entry directly:

```bash
yews init \
  --input config.toml \
  --output config.enc.toml \
  --format toml \
  --create-example
```

## Basic usage

Every command documents its full semantics in `--help` (English); this page only walks the happy paths.

### Encrypt configuration files

```bash
# encrypt every file registered in the config
yews encrypt

# encrypt one registered plaintext file, using the configured
# encrypted path and authorization
yews encrypt config.toml

# encrypt a single file to an explicit output path
yews encrypt config.toml -o config.enc.toml

# filter and encrypt within a configured group's directory scope
yews encrypt './configs/*.toml'
```

### Decrypt configuration files

```bash
# decrypt every file registered in the config
yews decrypt

# decrypt one registered encrypted file, using the configured
# plaintext path
yews decrypt config.enc.toml

# decrypt a single file to an explicit output path
yews decrypt config.enc.toml -o config.toml

# filter and decrypt within a configured group's directory scope
yews decrypt './configs/*.enc.toml'
```

By default, `decrypt` refuses to overwrite an existing plaintext file whose content differs; add `--force` to write anyway.

### Preview the selection

```bash
yews plan
yews plan './configs/*.toml'
yews plan --json
```

`plan` inspects and reports the config mappings, formats, and current authorization with its origins; group discovery takes the union of the plaintext and encrypted sides and nothing is written. It is not an operation dry run and does not verify that the current identity can decrypt.

### Edit an encrypted file

```bash
# use the default editor (VISUAL, then EDITOR; vi or notepad as fallback)
yews edit -f config.enc.toml

# pick a specific editor for one invocation
VISUAL="code --wait" yews edit -f config.enc.toml
```

There is no editor flag; the editor always comes from `VISUAL` or `EDITOR`.

### View an encrypted file

```bash
# print the decrypted plaintext to stdout
yews view config.enc.toml
```

### Compare differences

```bash
# compare a plaintext file with its encrypted counterpart
yews diff config.toml
```

## Example config

After initialization, `.yewseal.toml` looks roughly like this:

```toml
[recipients]
defaults = ["owner"]

[recipients.registry]
owner = "age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

[[encryption.files]]
plaintext = "config.toml"
encrypted = "config.enc.toml"
```

Groups can also let YewSeal scan batches of files by pattern; see [Configuration - group scanning](/guide/configuration#group-scanning).

## Next steps

Run `yews <command> --help` for the complete options and semantics of any command, or browse the generated [CLI reference](/references/yews). [Configuration](/guide/configuration) explains `.yewseal.toml`, `.sops.yaml`, and key provisioning; [Workflows](/guide/workflows) shows how the commands compose in daily use.
