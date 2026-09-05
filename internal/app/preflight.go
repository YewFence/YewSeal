package app

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"
)

type PreflightResult struct {
	Selection      config.ResolvedSelection
	IdentityBundle agekey.IdentityBundle
	Parallel       int
	Force          bool
	Verbose        bool
	MetadataPairs  []config.ResolvedFilePair
	MetadataScope  string
}

type PreflightPrintOptions struct {
	JSON    bool
	Verbose bool
}

func PreflightEncrypt(cfg *config.Config, req EncryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:              task.ModeEncrypt,
		Target:               req.Target,
		Output:               req.Output,
		OutputSet:            req.OutputSet,
		Format:               req.Format,
		Patterns:             req.Patterns,
		AllowEmptyTarget:     true,
		UseConfiguredDefault: true,
	})
	if err != nil {
		return PreflightResult{}, err
	}

	metadataPairs, metadataScope := metadataPairsForSelection(selection)
	return PreflightResult{
		Selection:     selection,
		Parallel:      req.Parallel,
		Verbose:       req.Verbose,
		MetadataPairs: metadataPairs,
		MetadataScope: metadataScope,
	}, nil
}

func PreflightDecrypt(cfg *config.Config, req DecryptRequest) (PreflightResult, error) {
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command:              task.ModeDecrypt,
		Target:               req.Target,
		Output:               req.Output,
		OutputSet:            req.OutputSet,
		Format:               req.Format,
		Patterns:             req.Patterns,
		AllowEmptyTarget:     true,
		UseConfiguredDefault: true,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	identityBundle, err := agekey.GetIdentityBundleWithFallback(req.KeyFile, cfg.GetKeyFile(""))
	if err != nil {
		return PreflightResult{}, err
	}

	metadataPairs, metadataScope := metadataPairsForSelection(selection)
	return PreflightResult{
		Selection:      selection,
		IdentityBundle: identityBundle,
		Parallel:       req.Parallel,
		Force:          req.Force,
		Verbose:        req.Verbose,
		MetadataPairs:  metadataPairs,
		MetadataScope:  metadataScope,
	}, nil
}

type PlanRequest struct {
	Output    string
	OutputSet bool
	Format    string
	Target    string
	Patterns  []string
	Parallel  int
	Verbose   bool
}

func PrintPlan(w io.Writer, cfg *config.Config, req PlanRequest, opts PreflightPrintOptions) error {
	selection, err := config.ResolvePlanSelection(cfg, config.SelectionOptions{
		Target:    req.Target,
		Output:    req.Output,
		OutputSet: req.OutputSet,
		Format:    req.Format,
		Patterns:  req.Patterns,
	})
	if err != nil {
		return err
	}
	metadataPairs, metadataScope := metadataPairsForSelection(selection)
	result := PreflightResult{
		Selection:     selection,
		Parallel:      req.Parallel,
		Verbose:       req.Verbose,
		MetadataPairs: metadataPairs,
		MetadataScope: metadataScope,
	}
	return PrintPreflight(w, cfg, result, opts)
}

func PrintPreflight(w io.Writer, cfg *config.Config, result PreflightResult, opts PreflightPrintOptions) error {
	if opts.JSON {
		return printPreflightJSON(w, cfg, result)
	}
	return printPreflightTable(w, cfg, result, opts)
}

func metadataPairsForSelection(selection config.ResolvedSelection) ([]config.ResolvedFilePair, string) {
	if selection.ConfigMode && len(selection.AllConfigPairs) > 0 {
		return selection.AllConfigPairs, "project"
	}
	return selection.FilePairs, "selection"
}

func printPreflightTable(w io.Writer, cfg *config.Config, result PreflightResult, opts PreflightPrintOptions) error {
	cwd := config.CurrentDir(cfg)
	selection := result.Selection
	scope := config.DisplayPath(cwd, selection.CurrentDirScope)
	if scope == "" {
		scope = "."
	}
	if _, err := fmt.Fprintf(w, "Loaded %d config files\n", len(selection.ConfigFiles)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Command %s\n", selection.Command); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scope %s\n", scope); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Selected %d file pairs\n", len(selection.FilePairs)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Metadata %s\n", result.MetadataScope); err != nil {
		return err
	}

	if opts.Verbose && len(selection.ConfigFiles) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "Config files"); err != nil {
			return err
		}
		for i, file := range selection.ConfigFiles {
			if _, err := fmt.Fprintf(w, "%d. %s\n", i+1, config.DisplayPath(cwd, file.Path)); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "Plaintext\tP.Source\tEncrypted\tE.Source\tFormat\tF.Source\tAliases\tRecipients\tAuthorization\tRegistry Sources\tSelected By"); err != nil {
		return err
	}
	for _, filePair := range selection.FilePairs {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			config.DisplayPath(cwd, filePair.PlaintextPath),
			config.FormatValueSource(filePair.PlaintextSource, cwd),
			config.DisplayPath(cwd, filePair.EncryptedPath),
			config.FormatValueSource(filePair.EncryptedSource, cwd),
			filePair.Format,
			config.FormatValueSource(filePair.FormatSource, cwd),
			strings.Join(filePair.RecipientAliases, ","),
			strings.Join(filePair.Recipients, ","),
			formatAuthorizationSource(filePair, cwd),
			formatRegistrySources(filePair.RecipientInfo, cwd),
			filePair.SelectedBy,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func printPreflightJSON(w io.Writer, cfg *config.Config, result PreflightResult) error {
	cwd := config.CurrentDir(cfg)
	payload := preflightJSON{
		Command:  result.Selection.Command,
		Scope:    config.DisplayPath(cwd, result.Selection.CurrentDirScope),
		Metadata: result.MetadataScope,
	}
	for _, file := range result.Selection.ConfigFiles {
		payload.ConfigFiles = append(payload.ConfigFiles, file.Path)
	}
	for _, filePair := range result.Selection.FilePairs {
		payload.FilePairs = append(payload.FilePairs, resolvedFilePairJSON(cwd, filePair))
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

type preflightJSON struct {
	Command     string              `json:"command"`
	Scope       string              `json:"scope,omitempty"`
	ConfigFiles []string            `json:"config_files"`
	Metadata    string              `json:"metadata"`
	FilePairs   []preflightPairJSON `json:"file_pairs"`
}

type preflightPairJSON struct {
	Plaintext        preflightPathJSON          `json:"plaintext"`
	Encrypted        preflightPathJSON          `json:"encrypted"`
	Format           preflightFormatJSON        `json:"format"`
	RecipientAliases []string                   `json:"recipient_aliases,omitempty"`
	Recipients       []string                   `json:"recipients,omitempty"`
	Authorization    preflightAuthorizationJSON `json:"authorization"`
	RecipientWarning string                     `json:"recipient_warning,omitempty"`
	SelectedBy       string                     `json:"selected_by"`
	Source           string                     `json:"source"`
}
type preflightAuthorizationJSON struct {
	Kind            string                   `json:"kind,omitempty"`
	Aliases         []string                 `json:"aliases,omitempty"`
	EffectiveSource preflightValueSourceJSON `json:"effective_source"`
	RegistrySources map[string]string        `json:"registry_sources,omitempty"`
}

type preflightPathJSON struct {
	Path    string                   `json:"path"`
	Display string                   `json:"display"`
	Source  preflightValueSourceJSON `json:"source"`
}

type preflightFormatJSON struct {
	Value  string                   `json:"value"`
	Source preflightValueSourceJSON `json:"source"`
}

type preflightValueSourceJSON struct {
	Kind       string `json:"kind"`
	ConfigPath string `json:"config,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

func formatAuthorizationSource(filePair config.ResolvedFilePair, cwd string) string {
	if filePair.RecipientInfo.EffectiveSource.Kind != "" {
		return config.FormatValueSource(filePair.RecipientInfo.EffectiveSource, cwd)
	}
	return filePair.RecipientInfo.Kind
}

func formatRegistrySources(info config.RecipientProvenance, cwd string) string {
	aliases := make([]string, 0, len(info.RegistrySources))
	for alias := range info.RegistrySources {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	formatted := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		formatted = append(formatted, alias+"="+config.DisplayPath(cwd, info.RegistrySources[alias]))
	}
	return strings.Join(formatted, ",")
}

func resolvedFilePairJSON(cwd string, filePair config.ResolvedFilePair) preflightPairJSON {
	return preflightPairJSON{
		Plaintext: preflightPathJSON{
			Path:    filePair.PlaintextPath,
			Display: config.DisplayPath(cwd, filePair.PlaintextPath),
			Source:  valueSourceJSON(filePair.PlaintextSource),
		},
		Encrypted: preflightPathJSON{
			Path:    filePair.EncryptedPath,
			Display: config.DisplayPath(cwd, filePair.EncryptedPath),
			Source:  valueSourceJSON(filePair.EncryptedSource),
		},
		Format: preflightFormatJSON{
			Value:  filePair.Format,
			Source: valueSourceJSON(filePair.FormatSource),
		},
		RecipientAliases: filePair.RecipientAliases,
		Recipients:       filePair.Recipients,
		Authorization: preflightAuthorizationJSON{
			Kind:            filePair.RecipientInfo.Kind,
			Aliases:         filePair.RecipientInfo.Aliases,
			EffectiveSource: valueSourceJSON(filePair.RecipientInfo.EffectiveSource),
			RegistrySources: filePair.RecipientInfo.RegistrySources,
		},
		RecipientWarning: filePair.RecipientWarning,
		SelectedBy:       filePair.SelectedBy,
		Source:           filePair.Source,
	}
}

func valueSourceJSON(source config.ValueSource) preflightValueSourceJSON {
	return preflightValueSourceJSON{
		Kind:       source.Kind,
		ConfigPath: source.ConfigPath,
		Detail:     source.Detail,
	}
}
