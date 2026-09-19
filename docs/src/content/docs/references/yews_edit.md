---
title: yews edit
---

Edit encrypted configuration file using SOPS

## Synopsis

Edit an encrypted file in place: decrypt it to a temporary file,
open an editor, then re-encrypt and write the result back on save.

The target must be registered in .yewseal.toml; either the encrypted
path or its configured plaintext path locates the mapping. There is no
default target and no --editor flag.

The editor is taken from VISUAL, then EDITOR; when both are unset or
empty, notepad is used on Windows and vi elsewhere. The variable may
carry the executable plus arguments using a restricted syntax, not a
full shell command:
  - spaces, tabs, and newlines separate arguments; single or double
    quotes keep a path or argument containing spaces together and
    adjacent pieces merge; empty arguments ("") are kept;
  - inside single quotes everything is literal; inside double quotes
    only \" and \\ lose their backslash; quote Windows paths;
  - outside quotes a backslash escapes the next character; unclosed
    quotes, a trailing backslash, NUL, invalid UTF-8, and unescaped
    | & ; < > ( ) $ or backticks are errors;
  - no environment variable, command substitution, glob, or ~ expansion
    happens and # is not a comment.
YewSeal spawns the editor process directly (no shell) and passes the
temporary file path as the final, separate argument.

The editor must exit only after the file is saved and closed (for
example "code --wait"), otherwise YewSeal re-encrypts before editing
finishes.

Exit codes: 0 on success (changed or unchanged); 1 when editing or
re-encryption fails; 2 for calling errors (no target, an unregistered
file, a missing or invalid .yewseal.toml, or an unusable identity
source).

Output: stdout stays empty; the update result (or "unchanged"),
warnings, and errors go to stderr.

See also: "yews view" to inspect a file read-only, "yews decrypt" to
write the plaintext to disk.

Documentation: https://yewfence.github.io/YewSeal/guide/tutorial

```
yews edit [flags]
```

## Examples

```
  # Edit a registered encrypted file
  yews edit -f config.enc.toml

  # The configured plaintext path locates the same mapping
  yews edit -f config.toml

  # Pick the editor per invocation
  VISUAL="code --wait" yews edit -f config.enc.toml
  VISUAL=vim yews edit -f config.enc.toml
```

## Options

```
  -f, --file string   Encrypted file to edit (must be registered in .yewseal.toml; its configured plaintext path also works) (env YEWSEAL_EDIT_FILE)
  -h, --help          help for edit
```

## Options inherited from parent commands

```
  -k, --key-file string   Path to the Age private key file (fallback: YEWSEAL_AGE_IDENTITIES, SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, then .age/keys.txt) (env YEWSEAL_KEY_FILE)
```

## SEE ALSO

* [yews](/references/yews/)	 - YewSeal - Encrypt/decrypt configuration files using SOPS and Age (supports TOML, YAML, JSON, ENV, INI, and binary files)
