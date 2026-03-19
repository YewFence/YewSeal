# NAME

yews - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

# SYNOPSIS

yews

```
[--help|-h]
[--key-file|-k]=[value]
[--version|-v]
```

**Usage**:

```
yews [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
```

# GLOBAL OPTIONS

**--help, -h**: show help

**--key-file, -k**="": Path to Age private key file

**--version, -v**: print the version


# COMMANDS

## init

Initialize project with Age keys and YewSeal config templates (single-file + batch)

**--create-example**: Create example file (non-interactive mode)

**--force, -f**: Force overwrite existing configuration

**--input, -i**="": Default input file for single-file mode (non-interactive mode)

**--output, -o**="": Default encrypted output file for single-file mode (non-interactive mode)

**--skip-sops-config**: Skip creating .sops.yaml file (non-interactive mode)

## encrypt, e

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini)

**--dir**="": Directory to scan for files (enables batch mode)

**--input, -i**="": Input file to encrypt (single file mode) (default: "wrangler.toml")

**--output, -o**="": Output encrypted file (single file mode only) (default: "wrangler.enc.toml.yaml")

**--output-dir**="": Output directory for encrypted files (batch mode)

**--output-suffix**="": Suffix for output files (batch mode) (default: ".enc.toml.yaml")

**--parallel, -P**="": Number of parallel workers for batch mode (default: 1)

**--pattern**="": Glob pattern for matching files in directory (default: "*.toml")

**--public-key, -p**="": Age public key for encryption

**--verbose, -v**: Enable verbose output

## decrypt, d

Decrypt encrypted file (output format determined by extension)

**--dir**="": Directory to scan for encrypted files (enables batch mode)

**--input, -i**="": Input encrypted file (single file mode) (default: "wrangler.enc.toml.yaml")

**--output, -o**="": Output decrypted file (single file mode only) (default: "wrangler.toml")

**--output-dir**="": Output directory for decrypted files (batch mode)

**--output-suffix**="": Suffix for output files (batch mode) (default: ".toml")

**--parallel, -P**="": Number of parallel workers for batch mode (default: 1)

**--pattern**="": Glob pattern for matching encrypted files (default: "*.enc.toml.yaml")

**--verbose, -v**: Enable verbose output

## edit

Edit encrypted configuration file using SOPS

**--editor, -e**="": Editor command to use (e.g., 'code -w', 'vim')

**--file, -f**="": Encrypted file to edit (default: "wrangler.enc.toml.yaml")

## check, doctor

Check if required external tools are installed

## sync

Sync sensitive files to secret management service

**--key-file, -k**="": Path to the key file to sync (default: ".age/keys.txt")

**--name, -n**="": Secret name in the provider (default: "AGE_KEY_FILE")

**--path**="": Path/folder in the provider (e.g., /yewseal)

**--provider, -p**="": Secret management provider (infisical) (default: "infisical")

## help, h

Shows a list of commands or help for one command
