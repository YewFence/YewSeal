package cli

import "github.com/spf13/pflag"

type encryptOptions struct {
	KeyFile        string `mapstructure:"key-file"`
	Output         string `mapstructure:"output"`
	Parallel       int    `mapstructure:"parallel"`
	Force          bool   `mapstructure:"force"`
	Verbose        bool   `mapstructure:"verbose"`
	SyncSOPSConfig bool   `mapstructure:"sync-sops-config"`
}

type decryptOptions struct {
	KeyFile  string `mapstructure:"key-file"`
	Output   string `mapstructure:"output"`
	Parallel int    `mapstructure:"parallel"`
	Force    bool   `mapstructure:"force"`
	Strict   bool   `mapstructure:"strict"`
	Verbose  bool   `mapstructure:"verbose"`
}

type planOptions struct {
	Verbose bool `mapstructure:"verbose"`
	JSON    bool `mapstructure:"json"`
}

type initOptions struct {
	Force          bool   `mapstructure:"force"`
	Input          string `mapstructure:"input"`
	Output         string `mapstructure:"output"`
	Format         string `mapstructure:"format"`
	CreateExample  bool   `mapstructure:"create-example"`
	SyncSOPSConfig bool   `mapstructure:"sync-sops-config"`
}

type editOptions struct {
	KeyFile string `mapstructure:"key-file"`
	File    string `mapstructure:"file"`
}

type viewOptions struct {
	KeyFile string `mapstructure:"key-file"`
	Verbose bool   `mapstructure:"verbose"`
}

type diffOptions struct {
	KeyFile string `mapstructure:"key-file"`
	Color   string `mapstructure:"color"`
	Verbose bool   `mapstructure:"verbose"`
}

func addEncryptFlags(flags *pflag.FlagSet, opts *encryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output encrypted file for a single file target")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Freshly encrypt every existing plaintext and rotate its data key")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.SyncSOPSConfig, "sync-sops-config", opts.SyncSOPSConfig, "Sync the complete project policy to .sops.yaml after encryption")
}

func addDecryptFlags(flags *pflag.FlagSet, opts *decryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output plaintext file for a single file target")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force overwrite existing plaintext file when it differs from decrypted content")
	flags.BoolVar(&opts.Strict, "strict", false, "Require every selected file to be decrypted")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output (selection info and per-file results on stderr)")
}

func addPlanFlags(flags *pflag.FlagSet, opts *planOptions) {
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.JSON, "json", false, "Print configured file mappings as JSON")
}
