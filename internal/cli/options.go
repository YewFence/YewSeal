package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/YewFence/YewSeal/internal/errx"
)

type optionBinding struct {
	flag *pflag.Flag
	env  string
}

type optionResolver struct {
	viper         *viper.Viper
	target        any
	bindings      []optionBinding
	bindErr       error
	inheritedDone bool
}

func newOptionResolver(cmd *cobra.Command, target any) *optionResolver {
	resolver := &optionResolver{viper: viper.New(), target: target}
	resolver.bindFlags(cmd.LocalNonPersistentFlags(), commandEnvPrefix(cmd.Name()))
	return resolver
}

func (r *optionResolver) before(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if !r.inheritedDone {
			r.bindFlags(cmd.InheritedFlags(), "YEWSEAL")
			r.inheritedDone = true
		}
		if r.bindErr != nil {
			return errx.Usage(r.bindErr)
		}
		if err := r.validateEnvironment(); err != nil {
			return errx.Usage(err)
		}
		if err := r.viper.Unmarshal(r.target); err != nil {
			return errx.Usage(fmt.Errorf("failed to resolve %s options: %w", cmd.Name(), err))
		}
		if validate == nil {
			return nil
		}
		return errx.Usage(validate(cmd, args))
	}
}

func (r *optionResolver) IsSet(name string) bool {
	return r.viper.IsSet(name)
}

func (r *optionResolver) bindFlags(flags *pflag.FlagSet, envPrefix string) {
	flags.VisitAll(func(flag *pflag.Flag) {
		envName := envName(envPrefix, flag.Name)
		if err := r.viper.BindPFlag(flag.Name, flag); err != nil {
			r.bindErr = errors.Join(r.bindErr, err)
			return
		}
		if err := r.viper.BindEnv(flag.Name, envName); err != nil {
			r.bindErr = errors.Join(r.bindErr, err)
			return
		}
		annotateFlagEnvironment(flag, envName)
		r.bindings = append(r.bindings, optionBinding{flag: flag, env: envName})
	})
}

func (r *optionResolver) validateEnvironment() error {
	var result error
	for _, binding := range r.bindings {
		if binding.flag.Changed {
			continue
		}
		value, ok := os.LookupEnv(binding.env)
		if !ok || value == "" {
			continue
		}
		var err error
		switch binding.flag.Value.Type() {
		case "bool":
			_, err = strconv.ParseBool(value)
		case "int":
			_, err = strconv.Atoi(value)
		}
		if err != nil {
			result = errors.Join(result, fmt.Errorf("invalid %s: expected %s", binding.env, environmentType(binding.flag)))
		}
	}
	return result
}

func environmentType(flag *pflag.Flag) string {
	switch flag.Value.Type() {
	case "bool":
		return "a boolean"
	case "int":
		return "an integer"
	default:
		return flag.Value.Type()
	}
}

func commandEnvPrefix(command string) string {
	return "YEWSEAL_" + envToken(command)
}

func envName(prefix, flagName string) string {
	return prefix + "_" + envToken(flagName)
}

func envToken(value string) string {
	return strings.ToUpper(strings.ReplaceAll(value, "-", "_"))
}

func annotateFlagEnvironment(flag *pflag.Flag, env string) {
	annotation := " (env " + env + ")"
	if !strings.Contains(flag.Usage, annotation) {
		flag.Usage += annotation
	}
}
