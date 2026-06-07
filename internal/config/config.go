package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultDecryptedFile = "wrangler.toml"
	defaultEncryptedFile = "wrangler.enc.toml.yaml"
	defaultKeyFile       = ".age/keys.txt"
)

// Config represents the YewSeal configuration.
type Config struct {
	Encryption EncryptionConfig `toml:"encryption"`
	Key        KeyConfig        `toml:"key"`
	Sync       SyncConfig       `toml:"sync"`

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
	Groups []GroupConfig `toml:"groups"`
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

	ConfigPath string `toml:"-"`
	ConfigDir  string `toml:"-"`
	Source     string `toml:"-"`
}

type GroupConfig struct {
	Patterns        []string `toml:"patterns"`
	FormatRules     []string `toml:"format_rules"`
	UnknownAsBinary bool     `toml:"unknown_as_binary"`

	ConfigPath string `toml:"-"`
	ConfigDir  string `toml:"-"`
}

// KeyConfig defines key file location.
type KeyConfig struct {
	// FilePath is the path to Age private key file.
	// Do NOT store the actual key value here to avoid leaking secrets.
	FilePath string `toml:"file_path"`
	// PublicKey is the Age public key for encryption (safe to commit).
	PublicKey string `toml:"public_key"`
}

// SyncConfig defines Age key synchronization settings.
type SyncConfig struct {
	// Provider is the secret management provider name.
	Provider string `toml:"provider,omitempty"`
	// ProjectID is the provider project identifier used by sync commands.
	ProjectID string `toml:"project_id,omitempty"`
	// SecretName is the remote secret name for the Age key file.
	SecretName string `toml:"secret_name,omitempty"`
	// Path is the remote path/folder in the provider.
	Path string `toml:"path,omitempty"`
	// Environment is the remote environment name in the provider.
	Environment string `toml:"environment,omitempty"`
}

// DefaultFilePair returns the default plaintext/encrypted mapping.
func DefaultFilePair() FilePair {
	return FilePair{
		PlaintextPath: defaultDecryptedFile,
		EncryptedPath: defaultEncryptedFile,
		Source:        "default",
	}
}

// DefaultConfig returns a config with default values.
func DefaultConfig() *Config {
	return &Config{
		Encryption: EncryptionConfig{
			Files: []FilePair{DefaultFilePair()},
		},
		Key: KeyConfig{
			FilePath: defaultKeyFile,
		},
	}
}

// LoadConfig loads configuration from .yewseal.toml.
// Searches in the following locations (in order):
// 1. .yewseal/.yewseal.toml
// 2. .config/.yewseal.toml
// 3. .yewseal.toml
// Returns default config if file doesn't exist.
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
		Key: KeyConfig{
			FilePath: defaultKeyFile,
		},
		CurrentDir: cwd,
		UserConfig: true,
	}

	for _, configFile := range configFiles {
		partial, err := loadConfigFile(configFile)
		if err != nil {
			return nil, err
		}
		mergeConfig(config, partial)
	}

	if len(config.Encryption.Files) == 0 && len(config.Encryption.Groups) == 0 {
		config.Encryption.Files = []FilePair{DefaultFilePair()}
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
	metadata, err := toml.DecodeFile(configFile.Path, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configFile.Path, err)
	}
	if metadata.IsDefined("encryption", "group") {
		return nil, fmt.Errorf("invalid encryption.group: use [[encryption.groups]] instead")
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
	for i := range config.Encryption.Files {
		config.Encryption.Files[i].ConfigPath = configPath
		config.Encryption.Files[i].ConfigDir = configDir
		config.Encryption.Files[i].Source = "exact"
		config.Encryption.Files[i].PlaintextPath = resolveConfigPath(configDir, config.Encryption.Files[i].PlaintextPath)
		config.Encryption.Files[i].EncryptedPath = resolveConfigPath(configDir, config.Encryption.Files[i].EncryptedPath)
	}
	for i := range config.Encryption.Groups {
		config.Encryption.Groups[i].ConfigPath = configPath
		config.Encryption.Groups[i].ConfigDir = configDir
	}
	return config, nil
}

func mergeConfig(dst, src *Config) {
	dst.LoadedFiles = append(dst.LoadedFiles, src.LoadedFiles...)

	if strings.TrimSpace(src.Key.FilePath) != "" {
		dst.Key.FilePath = resolveConfigPath(src.LoadedFiles[0].Dir, src.Key.FilePath)
	}
	if strings.TrimSpace(src.Key.PublicKey) != "" {
		dst.Key.PublicKey = src.Key.PublicKey
	}
	if strings.TrimSpace(src.Sync.Provider) != "" {
		dst.Sync.Provider = src.Sync.Provider
	}
	if strings.TrimSpace(src.Sync.ProjectID) != "" {
		dst.Sync.ProjectID = src.Sync.ProjectID
	}
	if strings.TrimSpace(src.Sync.SecretName) != "" {
		dst.Sync.SecretName = src.Sync.SecretName
	}
	if strings.TrimSpace(src.Sync.Path) != "" {
		dst.Sync.Path = src.Sync.Path
	}
	if strings.TrimSpace(src.Sync.Environment) != "" {
		dst.Sync.Environment = src.Sync.Environment
	}

	for _, filePair := range src.Encryption.Files {
		dst.Encryption.Files = upsertFilePair(dst.Encryption.Files, filePair)
	}
	dst.Encryption.Groups = append(dst.Encryption.Groups, src.Encryption.Groups...)
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

// GetKeyFile returns the key file path.
// Priority: provided value > config file > default.
func (c *Config) GetKeyFile(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Key.FilePath != "" {
		return c.Key.FilePath
	}
	return defaultKeyFile
}

// GetPublicKey returns the Age public key.
func (c *Config) GetPublicKey() string {
	return c.Key.PublicKey
}

// GetSyncProvider returns the key sync provider.
// Priority: provided value > config file > default.
func (c *Config) GetSyncProvider(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Sync.Provider != "" {
		return c.Sync.Provider
	}
	return "infisical"
}

// GetSyncSecretName returns the key sync secret name.
// Priority: provided value > config file > default.
func (c *Config) GetSyncSecretName(provided string) string {
	if provided != "" {
		return provided
	}
	if c.Sync.SecretName != "" {
		return c.Sync.SecretName
	}
	return "AGE_KEY_FILE"
}

// GetSyncProjectID returns the key sync project identifier.
// Priority: provided value > config file > default.
func (c *Config) GetSyncProjectID(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.ProjectID
}

// GetSyncPath returns the key sync remote path.
// Priority: provided value > config file > default.
func (c *Config) GetSyncPath(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.Path
}

// GetSyncEnvironment returns the key sync remote environment.
// Priority: provided value > config file > default.
func (c *Config) GetSyncEnvironment(provided string) string {
	if provided != "" {
		return provided
	}
	return c.Sync.Environment
}

// GetFiles returns configured file mappings.
func (c *Config) GetFiles() []FilePair {
	if len(c.Encryption.Files) == 0 {
		return []FilePair{DefaultFilePair()}
	}
	files := make([]FilePair, len(c.Encryption.Files))
	copy(files, c.Encryption.Files)
	return files
}

func (c *Config) GetGroups() []GroupConfig {
	groups := make([]GroupConfig, 0, len(c.Encryption.Groups))
	for _, group := range c.Encryption.Groups {
		groups = append(groups, GroupConfig{
			Patterns:        append([]string(nil), group.Patterns...),
			FormatRules:     append([]string(nil), group.FormatRules...),
			UnknownAsBinary: group.UnknownAsBinary,
			ConfigPath:      group.ConfigPath,
			ConfigDir:       group.ConfigDir,
		})
	}
	return groups
}

// GetPrimaryFilePair returns the first configured file mapping.
func (c *Config) GetPrimaryFilePair() FilePair {
	files := c.GetFiles()
	return files[0]
}
