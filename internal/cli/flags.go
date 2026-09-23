package cli

import "github.com/spf13/pflag"

type encryptOptions struct {
	KeyFile        string `mapstructure:"key-file"`
	Output         string `mapstructure:"output"`
	Parallel       int    `mapstructure:"parallel"`
	Force          bool   `mapstructure:"force"`
	Verbose        bool   `mapstructure:"verbose"`
	JSON           bool   `mapstructure:"json"`
	SyncSOPSConfig bool   `mapstructure:"sync-sops-config"`
}

type decryptOptions struct {
	KeyFile  string `mapstructure:"key-file"`
	Output   string `mapstructure:"output"`
	Parallel int    `mapstructure:"parallel"`
	Force    bool   `mapstructure:"force"`
	Strict   bool   `mapstructure:"strict"`
	Verbose  bool   `mapstructure:"verbose"`
	JSON     bool   `mapstructure:"json"`
}

type cleanOptions struct {
	KeyFile         string `mapstructure:"key-file"`
	Force           bool   `mapstructure:"force"`
	RemoveDifferent bool   `mapstructure:"remove-different"`
	SkipDifferent   bool   `mapstructure:"skip-different"`
	Verbose         bool   `mapstructure:"verbose"`
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
	JSON           bool   `mapstructure:"json"`
}

type editOptions struct {
	KeyFile string `mapstructure:"key-file"`
	File    string `mapstructure:"file"`
}

type viewOptions struct {
	KeyFile string `mapstructure:"key-file"`
	Verbose bool   `mapstructure:"verbose"`
	JSON    bool   `mapstructure:"json"`
}

type identitiesOptions struct {
	KeyFile string `mapstructure:"key-file"`
	JSON    bool   `mapstructure:"json"`
	Reveal  bool   `mapstructure:"reveal"`
}

type diffOptions struct {
	KeyFile string `mapstructure:"key-file"`
	Color   string `mapstructure:"color"`
	Verbose bool   `mapstructure:"verbose"`
	JSON    bool   `mapstructure:"json"`
}

func addEncryptFlags(flags *pflag.FlagSet, opts *encryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output encrypted file for a single file target")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Freshly encrypt every existing plaintext and rotate its data key")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.JSON, "json", false, "Print the batch report as JSON on stdout (diagnostics stay on stderr)")
	flags.BoolVar(&opts.SyncSOPSConfig, "sync-sops-config", opts.SyncSOPSConfig, "Sync the complete project policy to .sops.yaml after encryption")
	markSharedEnv(flags.Lookup("sync-sops-config"), syncSOPSConfigEnv)
}

func addDecryptFlags(flags *pflag.FlagSet, opts *decryptOptions) {
	flags.StringVarP(&opts.Output, "output", "o", opts.Output, "Output plaintext file for a single file target")
	flags.IntVarP(&opts.Parallel, "parallel", "P", opts.Parallel, "Number of parallel workers for batch mode (minimum 1)")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force overwrite existing plaintext file when it differs from decrypted content")
	flags.BoolVar(&opts.Strict, "strict", false, "Require every selected file to be decrypted")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output (selection info and per-file results on stderr)")
	flags.BoolVar(&opts.JSON, "json", false, "Print the batch report as JSON on stdout (diagnostics stay on stderr)")
}

func addCleanFlags(flags *pflag.FlagSet, opts *cleanOptions) {
	flags.BoolVar(&opts.Force, "force", false, "DANGEROUS: remove all selected plaintext without recoverability checks (CLI only)")
	markCLIOnlyFlag(flags.Lookup("force"))
	flags.BoolVar(&opts.RemoveDifferent, "remove-different", false, "Remove plaintext that differs from successfully decrypted ciphertext without prompting")
	flags.BoolVar(&opts.SkipDifferent, "skip-different", false, "Keep plaintext that differs from successfully decrypted ciphertext without prompting")
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output (selection info and already-absent results on stderr)")
}

func addPlanFlags(flags *pflag.FlagSet, opts *planOptions) {
	flags.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	flags.BoolVar(&opts.JSON, "json", false, "Print configured file mappings as JSON")
}

type verifyOptions struct {
	KeyFile        string `mapstructure:"key-file"`
	Decrypt        bool   `mapstructure:"decrypt"`
	NoDecrypt      bool   `mapstructure:"no-decrypt"`
	SyncSOPSConfig bool   `mapstructure:"sync-sops-config"`
	JSON           bool   `mapstructure:"json"`
	Verbose        bool   `mapstructure:"verbose"`
}
