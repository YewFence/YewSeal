## [0.12.1] - 2026-09-17

### ⚙️ Miscellaneous Tasks

- *(gomod-deps)* Update go modules (#31)
- *(go-deps)* Update go module directive to v1.27.1 (#32)
- *(renovate)* Exempt sops fork from minimum release age
- *(gomod-deps)* Update module github.com/yewfence/sops/v3 to v3.0.0-20260917010022-e81f46a61f75 (#35)
## [0.12.0] - 2026-09-17

### 🚀 Features

- *(skills)* Add YewSeal agent skills for CLI users (#28)
- *(encrypt)* Preserve unchanged ciphertext (#30)
## [0.11.0] - 2026-09-15

### 🐛 Bug Fixes

- *(docs)* Rebase root-absolute Markdown links with the base path

### 🚜 Refactor

- Migrate to Astro
- *(cli)* [**breaking**] Drop diff --strict and align diff patterns with the plaintext side

### 📚 Documentation

- English documentation and self-documenting CLI help
- [**breaking**] Restructure site into tutorial-driven guides
- Translate schema comments to English
- State the shared glob dialect precisely in target selection
## [0.10.0] - 2026-09-14

### 🚜 Refactor

- *(config)* [**breaking**] Require group patterns and drop '#' pattern syntax (#24)
## [0.9.0] - 2026-09-11

### 🚀 Features

- *(cli)* [**breaking**] Identity skip and strict processing mode (#19)

### 🐛 Bug Fixes

- Prevent recursive encryption of ciphertext files (#15)

### 🚜 Refactor

- [**breaking**] Remove built-in key sync and project key-path config (#16)
- *(cli)* [**breaking**] Tighten config loading timing and remove format overrides (#17)
- *(selection)* [**breaking**] Unify file selection pipeline and clarify plan semantics (#20)
- *(read)* Unify view and diff read paths (#21)
- *(cli)* [**breaking**] Unify command output and diagnostics handling (#23)
## [0.8.0] - 2026-09-05

### 🚀 Features

- [**breaking**] Recipient authorization (#12)
- *(edit)* [**breaking**] Replace --editor flag with VISUAL/EDITOR and safe parsing
- Add renovate conf

### ⚙️ Miscellaneous Tasks

- Bump go deps
## [0.7.0] - 2026-08-23

### 🚀 Features

- *(schema)* Add CUE authoritative schema for .yewseal.toml with tripwire sync tests

### 🐛 Bug Fixes

- *(cd)* Add release-package.sh

### 🚜 Refactor

- *(toml)* [**breaking**] Use native TOML store from YewFence/sops fork
- [**breaking**] Remove `check`/`doctor` command and `internal/doctor` package
- Migrate TOML handling from BurntSushi/toml to pelletier/go-toml/v2
- *(cli)* Extract commands into internal/cli package and replace docs generation
- Native toml (#8)

### 📚 Documentation

- Restructure documentation and update for native TOML encryption
- Correct key priority order and improve secret setup instructions

### 🧪 Testing

- *(gendocs)* Verify generated output matches committed references

### ⚙️ Miscellaneous Tasks

- Update to my new template
- *(release)* Stop breaking changes from bumping major during 0.x
- Release v0.7.0 (#9)
## [0.7.0] - 2026-08-23

### 🚀 Features

- *(schema)* Add CUE authoritative schema for .yewseal.toml with tripwire sync tests

### 🚜 Refactor

- *(toml)* [**breaking**] Use native TOML store from YewFence/sops fork
- [**breaking**] Remove `check`/`doctor` command and `internal/doctor` package
- Migrate TOML handling from BurntSushi/toml to pelletier/go-toml/v2
- *(cli)* Extract commands into internal/cli package and replace docs generation
- Native toml (#8)

### 📚 Documentation

- Restructure documentation and update for native TOML encryption
- Correct key priority order and improve secret setup instructions

### 🧪 Testing

- *(gendocs)* Verify generated output matches committed references

### ⚙️ Miscellaneous Tasks

- Update to my new template
- *(release)* Stop breaking changes from bumping major during 0.x
