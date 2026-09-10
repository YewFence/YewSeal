package cli

import (
	"os"

	"github.com/spf13/pflag"
)

type encryptOptions struct {
	Output   string
	Patterns []string
	Parallel int
	Verbose  bool
}

type decryptOptions struct {
	Output   string
	Patterns []string
	Parallel int
	Force    bool
	Verbose  bool
}

type planOptions struct {
	Output   string
	Patterns []string
	Parallel int
	Verbose  bool
	JSON     bool
}

func addEncryptFlags(flags *pflag.FlagSet, opts *encryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output encrypted file for a single file target")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Pattern filter for configured groups")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
}

func addDecryptFlags(flags *pflag.FlagSet, opts *decryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output plaintext file for a single file target")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Pattern filter for configured groups")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force overwrite existing plaintext file when it differs from decrypted content")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
}

func addPlanFlags(flags *pflag.FlagSet, opts *planOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output file for a single file target")
	flags.StringSliceVar(&opts.Patterns, "pattern", nil, "Group pattern for directory mode or encryption.groups override")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.JSON, "json", false, "Print preflight result as JSON")
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
