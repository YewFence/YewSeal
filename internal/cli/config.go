package cli

import (
	"fmt"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/spf13/cobra"
)

type configLoader func() (*config.Config, error)

// Cobra validates Args before RunE, so config is loaded only for valid business invocations.
func withConfig(load configLoader, run func(*cobra.Command, []string, *config.Config) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := load()
		if err != nil {
			return errx.Usage(fmt.Errorf("failed to load config: %w", err))
		}
		return run(cmd, args, cfg)
	}
}
