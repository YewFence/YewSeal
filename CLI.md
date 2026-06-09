## yews

YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

### Options

```
  -h, --help              help for yews
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews check](#yews-check)	 - Check if required external tools are installed
* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell
* [yews decrypt](#yews-decrypt)	 - Decrypt encrypted file (output format determined by extension)
* [yews diff](#yews-diff)	 - Compare plaintext file with decrypted encrypted file
* [yews edit](#yews-edit)	 - Edit encrypted configuration file using SOPS
* [yews encrypt](#yews-encrypt)	 - Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)
* [yews init](#yews-init)	 - Initialize project with Age keys and YewSeal config entries
* [yews plan](#yews-plan)	 - Run preflight and print the resolved file selection without writing files
* [yews sync](#yews-sync)	 - Sync sensitive files to secret management service
* [yews view](#yews-view)	 - Print decrypted plaintext to standard output without writing files

## yews check

Check if required external tools are installed

```
yews check [flags]
```

### Options

```
  -h, --help   help for check
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews completion

Generate the autocompletion script for the specified shell

### Synopsis

Generate the autocompletion script for yews for the specified shell.
See each sub-command's help for details on how to use the generated script.


### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
* [yews completion bash](#yews-completion-bash)	 - Generate the autocompletion script for bash
* [yews completion fish](#yews-completion-fish)	 - Generate the autocompletion script for fish
* [yews completion powershell](#yews-completion-powershell)	 - Generate the autocompletion script for powershell
* [yews completion zsh](#yews-completion-zsh)	 - Generate the autocompletion script for zsh

## yews completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(yews completion bash)

To load completions for every new session, execute once:

#### Linux:

	yews completion bash > /etc/bash_completion.d/yews

#### macOS:

	yews completion bash > $(brew --prefix)/etc/bash_completion.d/yews

You will need to start a new shell for this setup to take effect.


```
yews completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell

## yews completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	yews completion fish | source

To load completions for every new session, execute once:

	yews completion fish > ~/.config/fish/completions/yews.fish

You will need to start a new shell for this setup to take effect.


```
yews completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell

## yews completion help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type completion help [path to command] for full details.

```
yews completion help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell

## yews completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	yews completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
yews completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell

## yews completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(yews completion zsh)

To load completions for every new session, execute once:

#### Linux:

	yews completion zsh > "${fpath[1]}/_yews"

#### macOS:

	yews completion zsh > $(brew --prefix)/share/zsh/site-functions/_yews

You will need to start a new shell for this setup to take effect.


```
yews completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews completion](#yews-completion)	 - Generate the autocompletion script for the specified shell

## yews decrypt

Decrypt encrypted file (output format determined by extension)

```
yews decrypt [command options] [path] [flags]
```

### Options

```
  -f, --force                 Force overwrite existing plaintext file when it differs from decrypted content
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for decrypt
  -o, --output string         Output plaintext file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Group pattern for temporary directory mode or encryption.groups override
      --unknown-as-binary     Allow group mode to treat unknown encrypted inputs as binary when needed
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews diff

Compare plaintext file with decrypted encrypted file

```
yews diff [target] [flags]
```

### Options

```
      --color string    Colorize diff output (auto/always/never) (default "auto")
      --format string   Format override for the selected target (toml/yaml/json/env/ini/binary)
  -h, --help            help for diff
  -v, --verbose         Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews edit

Edit encrypted configuration file using SOPS

```
yews edit [flags]
```

### Options

```
  -e, --editor string   Editor command to use (e.g., 'code -w', 'vim')
  -f, --file string     Encrypted file to edit (default "wrangler.enc.toml.yaml")
  -h, --help            help for edit
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews encrypt

Encrypt configuration file (supports .toml, .yaml, .yml, .json, .env, .ini, and binary output)

```
yews encrypt [command options] [path] [flags]
```

### Options

```
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for encrypt
  -o, --output string         Output encrypted file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Group pattern for temporary directory mode or encryption.groups override
  -p, --public-key string     Age public key for encryption
      --unknown-as-binary     Allow group mode to encrypt unknown plaintext formats as binary
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type yews help [path to command] for full details.

```
yews help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews init

Initialize project with Age keys and YewSeal config entries

```
yews init [flags]
```

### Options

```
      --create-example     Create example file (non-interactive mode)
  -f, --force              Force overwrite existing configuration
      --format string      Format override for the first config entry (toml/yaml/json/env/ini/binary)
  -h, --help               help for init
  -i, --input string       Plaintext file for the first config entry (non-interactive mode)
  -o, --output string      Encrypted file for the first config entry (non-interactive mode)
      --skip-sops-config   Skip creating .sops.yaml file (non-interactive mode)
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews plan

Run preflight and print the resolved file selection without writing files

```
yews plan [command options] [path] [flags]
```

### Options

```
      --format string         Format override for file targets (toml/yaml/json/env/ini/binary)
      --format-rule strings   Group format rule in <pattern>=<format> form
  -h, --help                  help for plan
      --json                  Print preflight result as JSON
  -o, --output string         Output file for a single file target
  -P, --parallel int          Number of parallel workers for batch mode (default 1)
      --pattern strings       Group pattern for directory mode or encryption.groups override
      --unknown-as-binary     Allow group mode to treat unknown formats as binary
  -v, --verbose               Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

## yews sync

Sync sensitive files to secret management service

```
yews sync [flags]
```

### Options

```
      --env string          Environment name in the provider
  -h, --help                help for sync
  -k, --key-file string     Path to the key file to sync (default ".age/keys.txt")
  -n, --name string         Secret name in the provider (default "AGE_KEY_FILE")
      --path string         Path/folder in the provider (e.g., /yewseal)
      --project-id string   Infisical project ID
  -p, --provider string     Secret management provider (infisical) (default "infisical")
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)
* [yews sync pull](#yews-sync-pull)	 - Pull key from secret management service to local file

## yews sync help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type sync help [path to command] for full details.

```
yews sync help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews sync](#yews-sync)	 - Sync sensitive files to secret management service

## yews sync pull

Pull key from secret management service to local file

```
yews sync pull [flags]
```

### Options

```
      --env string          Environment name in the provider
  -h, --help                help for pull
  -k, --key-file string     Local path to save the key file (default ".age/keys.txt")
  -n, --name string         Secret name in the provider (default "AGE_KEY_FILE")
      --path string         Path/folder in the provider (e.g., /yewseal)
      --project-id string   Infisical project ID
  -p, --provider string     Secret management provider (infisical) (default "infisical")
```

### SEE ALSO

* [yews sync](#yews-sync)	 - Sync sensitive files to secret management service

## yews view

Print decrypted plaintext to standard output without writing files

```
yews view [command options] <target> [flags]
```

### Options

```
      --format string   Format override for the selected target (toml/yaml/json/env/ini/binary)
  -h, --help            help for view
  -v, --verbose         Enable verbose output
```

### Options inherited from parent commands

```
  -k, --key-file string   Path to Age private key file
```

### SEE ALSO

* [yews](#yews)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI)

