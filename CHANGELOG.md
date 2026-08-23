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
