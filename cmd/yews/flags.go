package main

import (
	"os"

	"github.com/spf13/pflag"
)

type encryptOptions struct {
	Output          string
	Format          string
	Patterns        []string
	FormatRules     []string
	UnknownAsBinary bool
	Parallel        int
	PublicKey       string
	Verbose         bool
}

type decryptOptions struct {
	Output          string
	Format          string
	Patterns        []string
	FormatRules     []string
	UnknownAsBinary bool
	Parallel        int
	Force           bool
	Verbose         bool
}

type planOptions struct {
	Output          string
	Format          string
	Patterns        []string
	FormatRules     []string
	UnknownAsBinary bool
	Parallel        int
	Verbose         bool
	JSON            bool
}

type syncOptions struct {
	KeyFile   string
	Name      string
	ProjectID string
	Path      string
	Env       string
	Provider  string
}

func addEncryptFlags(flags *pflag.FlagSet, opts *encryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output encrypted file for a single file target")
	flags.StringVar(&opts.Format, "format", opts.Format, "Format override for file targets (toml/yaml/json/env/ini/binary)")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Group pattern for temporary directory mode or encryption.groups override")
	flags.StringSliceVar(&opts.FormatRules, "format-rule", nil, "Group format rule in <pattern>=<format> form")
	flags.BoolVar(&opts.UnknownAsBinary, "unknown-as-binary", false, "Allow group mode to encrypt unknown plaintext formats as binary")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode")
	flags.StringVarP(&opts.PublicKey, "public-key", "p", opts.PublicKey, "Age public key for encryption")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
}

func addDecryptFlags(flags *pflag.FlagSet, opts *decryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output plaintext file for a single file target")
	flags.StringVar(&opts.Format, "format", opts.Format, "Format override for file targets (toml/yaml/json/env/ini/binary)")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Group pattern for temporary directory mode or encryption.groups override")
	flags.StringSliceVar(&opts.FormatRules, "format-rule", nil, "Group format rule in <pattern>=<format> form")
	flags.BoolVar(&opts.UnknownAsBinary, "unknown-as-binary", false, "Allow group mode to treat unknown encrypted inputs as binary when needed")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force overwrite existing plaintext file when it differs from decrypted content")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
}

func addPlanFlags(flags *pflag.FlagSet, opts *planOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output file for a single file target")
	flags.StringVar(&opts.Format, "format", opts.Format, "Format override for file targets (toml/yaml/json/env/ini/binary)")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Group pattern for directory mode or encryption.groups override")
	flags.StringSliceVar(&opts.FormatRules, "format-rule", nil, "Group format rule in <pattern>=<format> form")
	flags.BoolVar(&opts.UnknownAsBinary, "unknown-as-binary", false, "Allow group mode to treat unknown formats as binary")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.JSON, "json", false, "Print preflight result as JSON")
}

func addSyncFlags(flags *pflag.FlagSet, opts *syncOptions) {
	flags.StringVarP(&opts.KeyFile, "key-file", "k", opts.KeyFile, "Path to the key file to sync")
	flags.StringVarP(&opts.Name, "name", "n", opts.Name, "Secret name in the provider")
	flags.StringVar(&opts.ProjectID, "project-id", "", "Infisical project ID")
	flags.StringVar(&opts.Path, "path", "", "Path/folder in the provider (e.g., /yewseal)")
	flags.StringVar(&opts.Env, "env", "", "Environment name in the provider")
	flags.StringVarP(&opts.Provider, "provider", "p", opts.Provider, "Secret management provider (infisical)")
}

func addSyncPullFlags(flags *pflag.FlagSet, opts *syncOptions) {
	flags.StringVarP(&opts.KeyFile, "key-file", "k", opts.KeyFile, "Local path to save the key file")
	flags.StringVarP(&opts.Name, "name", "n", opts.Name, "Secret name in the provider")
	flags.StringVar(&opts.ProjectID, "project-id", "", "Infisical project ID")
	flags.StringVar(&opts.Path, "path", "", "Path/folder in the provider (e.g., /yewseal)")
	flags.StringVar(&opts.Env, "env", "", "Environment name in the provider")
	flags.StringVarP(&opts.Provider, "provider", "p", opts.Provider, "Secret management provider (infisical)")
}

func flagChangedOrEnvSet(flags *pflag.FlagSet, name string, envNames ...string) bool {
	if flags.Changed(name) {
		return true
	}
	for _, name := range envNames {
		if _, ok := os.LookupEnv(name); ok {
			return true
		}
	}
	return false
}

func envValue(names ...string) string {
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
	}
	return ""
}
