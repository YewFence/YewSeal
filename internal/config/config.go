package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Config represents the YewSeal configuration.
type Config struct {
	Encryption EncryptionConfig `toml:"encryption"`
	Recipients RecipientConfig  `toml:"recipients"`

	LoadedFiles []LoadedFile `toml:"-"`
	CurrentDir  string       `toml:"-"`
	UserConfig  bool         `toml:"-"`
}

type LoadedFile struct {
	Path string
	Dir  string
}

// EncryptionConfig defines encrypted file mappings.
type EncryptionConfig struct {
	Files  []FilePair    `toml:"files"`
	Groups []GroupConfig `toml:"groups,omitempty"`
}

// FilePair defines one plaintext/encrypted file mapping.
type FilePair struct {
	// PlaintextPath is the plaintext file path used by encrypt input and decrypt output.
	PlaintextPath string `toml:"plaintext"`
	// EncryptedPath is the encrypted file path used by encrypt output and decrypt input.
	EncryptedPath string `toml:"encrypted"`
	// Format overrides the file format detection (toml/yaml/json/env/ini/binary).
	// Useful for files with non-standard extensions like .dev.vars.
	Format string `toml:"format,omitempty"`
	// Recipients is the explicit authorization set. A nil pointer means the field was omitted.
	Recipients *[]string `toml:"recipients,omitempty"`

	ConfigPath      string      `toml:"-"`
	ConfigDir       string      `toml:"-"`
	Source          string      `toml:"-"`
	RecipientSource ValueSource `toml:"-"`
}

// RecipientConfig defines the public authorization policy.
type RecipientConfig struct {
	// Defaults is the optional default alias set. A nil pointer means omitted.
	Defaults *[]string `toml:"defaults,omitempty"`
	// Registry maps stable aliases to public Age recipients.
	Registry map[string]string `toml:"registry,omitempty"`

	DefaultsConfigPath string            `toml:"-"`
	RegistrySources    map[string]string `toml:"-"`
}

type GroupConfig struct {
	Patterns        []string `toml:"patterns"`
	FormatRules     []string `toml:"format_rules"`
	UnknownAsBinary bool     `toml:"unknown_as_binary"`
	// Recipients is the optional group authorization set.
	Recipients *[]string `toml:"recipients,omitempty"`

	ConfigPath      string      `toml:"-"`
	ConfigDir       string      `toml:"-"`
	RecipientSource ValueSource `toml:"-"`
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{}
}

// LoadConfig loads configuration from .yewseal.toml.
// Searches in the following locations (in order):
// 1. .yewseal/.yewseal.toml
// 2. .config/.yewseal.toml
// 3. .yewseal.toml
// If no file exists, it returns an empty selection config.
func LoadConfig() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	configFiles, err := discoverConfigFiles(cwd)
	if err != nil {
		return nil, err
	}

	if len(configFiles) == 0 {
		config := DefaultConfig()
		config.CurrentDir = cwd
		return config, nil
	}

	config := &Config{
		CurrentDir: cwd,
		UserConfig: true,
	}

	for _, configFile := range configFiles {
		partial, err := loadConfigFile(configFile)
		if err != nil {
			return nil, err
		}
		if err := mergeConfig(config, partial); err != nil {
			return nil, err
		}
	}
	if hasRecipientPolicy(config) {
		if err := config.ValidateRecipientConfig(); err != nil {
			return nil, err
		}
	}

	for i, filePair := range config.Encryption.Files {
		if strings.TrimSpace(filePair.PlaintextPath) == "" {
			return nil, fmt.Errorf("invalid encryption.files[%d]: plaintext is required", i)
		}
		if strings.TrimSpace(filePair.EncryptedPath) == "" {
			return nil, fmt.Errorf("invalid encryption.files[%d]: encrypted is required", i)
		}
	}
	for groupIndex, group := range config.Encryption.Groups {
		for ruleIndex, rule := range group.FormatRules {
			if strings.TrimSpace(rule) == "" {
				return nil, fmt.Errorf("invalid encryption.groups[%d].format_rules[%d]: rule is empty", groupIndex, ruleIndex)
			}
		}
	}

	return config, nil
}

func discoverConfigFiles(cwd string) ([]LoadedFile, error) {
	searchDirs, err := configSearchDirs(cwd)
	if err != nil {
		return nil, err
	}

	files := make([]LoadedFile, 0, len(searchDirs))
	for _, dir := range searchDirs {
		configPath, err := highestPriorityConfigPath(dir)
		if err != nil {
			return nil, err
		}
		if configPath == "" {
			continue
		}
		files = append(files, LoadedFile{Path: configPath, Dir: dir})
	}
	return files, nil
}

func configSearchDirs(cwd string) ([]string, error) {
	root, ok, err := gitRoot(cwd)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []string{cwd}, nil
	}

	rel, err := filepath.Rel(root, cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current directory relative to git root: %w", err)
	}
	dirs := []string{root}
	if rel == "." {
		return dirs, nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		dirs = append(dirs, filepath.Join(dirs[len(dirs)-1], part))
	}
	return dirs, nil
}

func gitRoot(cwd string) (string, bool, error) {
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("failed to stat git marker %s: %w", gitPath, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}

func highestPriorityConfigPath(dir string) (string, error) {
	configPaths := []string{
		filepath.Join(dir, ".yewseal", ".yewseal.toml"),
		filepath.Join(dir, ".config", ".yewseal.toml"),
		filepath.Join(dir, ".yewseal.toml"),
	}
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat config file %s: %w", path, err)
		}
	}
	return "", nil
}

func loadConfigFile(configFile LoadedFile) (*Config, error) {
	config := &Config{}
	data, err := os.ReadFile(configFile.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFile.Path, err)
	}
	// Probe for removed fields to report actionable migration errors.
	var probe struct {
		Encryption struct {
			Group any `toml:"group"`
		} `toml:"encryption"`
		Key struct {
			PublicKey any `toml:"public_key"`
			FilePath  any `toml:"file_path"`
		} `toml:"key"`
	}
	if err := toml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFile.Path, err)
	}
	if probe.Encryption.Group != nil {
		return nil, fmt.Errorf("invalid encryption.group: use [[encryption.groups]] instead")
	}
	if probe.Key.PublicKey != nil {
		return nil, fmt.Errorf("deprecated key.public_key is not supported; use recipients.registry and recipients.defaults")
	}
	if probe.Key.FilePath != nil {
		return nil, fmt.Errorf("key.file_path is no longer supported; use --key-file or SOPS_AGE_KEY_FILE instead")
	}
	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFile.Path, err)
	}

	configPath, err := filepath.Abs(configFile.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config file %s: %w", configFile.Path, err)
	}
	configDir, err := filepath.Abs(configFile.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config root %s: %w", configFile.Dir, err)
	}
	configDir = filepath.Clean(configDir)
	config.LoadedFiles = []LoadedFile{{Path: configPath, Dir: configDir}}
	if config.Recipients.Defaults != nil {
		config.Recipients.DefaultsConfigPath = configPath
	}
	config.Recipients.RegistrySources = make(map[string]string, len(config.Recipients.Registry))
	for alias := range config.Recipients.Registry {
		config.Recipients.RegistrySources[alias] = configPath
	}
	for i := range config.Encryption.Files {
		config.Encryption.Files[i].ConfigPath = configPath
		config.Encryption.Files[i].ConfigDir = configDir
		config.Encryption.Files[i].Source = "exact"
		if config.Encryption.Files[i].Recipients != nil {
			config.Encryption.Files[i].RecipientSource = ValueSource{Kind: "file", ConfigPath: configPath, Detail: "recipients"}
		}
		config.Encryption.Files[i].PlaintextPath = resolveConfigPath(configDir, config.Encryption.Files[i].PlaintextPath)
		config.Encryption.Files[i].EncryptedPath = resolveConfigPath(configDir, config.Encryption.Files[i].EncryptedPath)
	}
	for i := range config.Encryption.Groups {
		config.Encryption.Groups[i].ConfigPath = configPath
		config.Encryption.Groups[i].ConfigDir = configDir
		if config.Encryption.Groups[i].Recipients != nil {
			config.Encryption.Groups[i].RecipientSource = ValueSource{Kind: "group", ConfigPath: configPath, Detail: "recipients"}
		}
	}
	return config, nil
}

func mergeConfig(dst, src *Config) error {
	dst.LoadedFiles = append(dst.LoadedFiles, src.LoadedFiles...)

	if src.Recipients.Defaults != nil {
		if dst.Recipients.Defaults != nil {
			return fmt.Errorf("recipient defaults are defined more than once")
		}
		defaults := append([]string(nil), (*src.Recipients.Defaults)...)
		dst.Recipients.Defaults = &defaults
		dst.Recipients.DefaultsConfigPath = src.Recipients.DefaultsConfigPath
	}
	if dst.Recipients.Registry == nil {
		dst.Recipients.Registry = make(map[string]string)
	}
	if dst.Recipients.RegistrySources == nil {
		dst.Recipients.RegistrySources = make(map[string]string)
	}
	for alias, recipient := range src.Recipients.Registry {
		if _, exists := dst.Recipients.Registry[alias]; exists {
			return fmt.Errorf("duplicate recipient alias %q", alias)
		}
		dst.Recipients.Registry[alias] = recipient
		if src.Recipients.RegistrySources != nil {
			dst.Recipients.RegistrySources[alias] = src.Recipients.RegistrySources[alias]
		}
	}

	for _, filePair := range src.Encryption.Files {
		dst.Encryption.Files = upsertFilePair(dst.Encryption.Files, filePair)
	}
	dst.Encryption.Groups = append(dst.Encryption.Groups, src.Encryption.Groups...)
	return nil
}

func upsertFilePair(files []FilePair, next FilePair) []FilePair {
	nextPlaintext := cleanAbsPath(next.PlaintextPath)
	nextEncrypted := cleanAbsPath(next.EncryptedPath)
	filtered := files[:0]
	for _, existing := range files {
		if cleanAbsPath(existing.PlaintextPath) == nextPlaintext || cleanAbsPath(existing.EncryptedPath) == nextEncrypted {
			continue
		}
		filtered = append(filtered, existing)
	}
	return append(filtered, next)
}

func resolveConfigPath(root, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(root, path))
}

func cleanAbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

// GetFiles returns configured file mappings.
func (c *Config) GetFiles() []FilePair {
	files := make([]FilePair, 0, len(c.Encryption.Files))
	for _, filePair := range c.Encryption.Files {
		filePair.Recipients = cloneOptionalStrings(filePair.Recipients)
		files = append(files, filePair)
	}
	return files
}

func (c *Config) GetGroups() []GroupConfig {
	groups := make([]GroupConfig, 0, len(c.Encryption.Groups))
	for _, group := range c.Encryption.Groups {
		groups = append(groups, GroupConfig{
			Patterns:        append([]string(nil), group.Patterns...),
			FormatRules:     append([]string(nil), group.FormatRules...),
			UnknownAsBinary: group.UnknownAsBinary,
			Recipients:      cloneOptionalStrings(group.Recipients),
			ConfigPath:      group.ConfigPath,
			ConfigDir:       group.ConfigDir,
			RecipientSource: group.RecipientSource,
		})
	}
	return groups
}
