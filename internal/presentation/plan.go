package presentation

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/YewFence/YewSeal/internal/config"
)

type PlanPrintOptions struct {
	JSON    bool
	Verbose bool
}

func (o *Output) Plan(cfg *config.Config, selection config.ResolvedSelection, opts PlanPrintOptions) error {
	if opts.JSON {
		return printPlanJSON(o, cfg, selection)
	}
	return printPlanTable(o, cfg, selection, opts)
}

func printPlanTable(w io.Writer, cfg *config.Config, selection config.ResolvedSelection, opts PlanPrintOptions) error {
	cwd := config.CurrentDir(cfg)
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

func printPlanJSON(w io.Writer, cfg *config.Config, selection config.ResolvedSelection) error {
	cwd := config.CurrentDir(cfg)
	payload := planJSON{
		Command: selection.Command,
		Scope:   config.DisplayPath(cwd, selection.CurrentDirScope),
	}
	for _, file := range selection.ConfigFiles {
		payload.ConfigFiles = append(payload.ConfigFiles, file.Path)
	}
	for _, filePair := range selection.FilePairs {
		payload.FilePairs = append(payload.FilePairs, resolvedFilePairJSON(cwd, filePair))
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

type planJSON struct {
	Command     string         `json:"command"`
	Scope       string         `json:"scope,omitempty"`
	ConfigFiles []string       `json:"config_files"`
	FilePairs   []planPairJSON `json:"file_pairs"`
}

type planPairJSON struct {
	Plaintext        planPathJSON          `json:"plaintext"`
	Encrypted        planPathJSON          `json:"encrypted"`
	Format           planFormatJSON        `json:"format"`
	RecipientAliases []string              `json:"recipient_aliases,omitempty"`
	Recipients       []string              `json:"recipients,omitempty"`
	Authorization    planAuthorizationJSON `json:"authorization"`
	RecipientWarning string                `json:"recipient_warning,omitempty"`
	SelectedBy       string                `json:"selected_by"`
	Source           string                `json:"source"`
}
type planAuthorizationJSON struct {
	Kind            string              `json:"kind,omitempty"`
	Aliases         []string            `json:"aliases,omitempty"`
	EffectiveSource planValueSourceJSON `json:"effective_source"`
	RegistrySources map[string]string   `json:"registry_sources,omitempty"`
}

type planPathJSON struct {
	Path    string              `json:"path"`
	Display string              `json:"display"`
	Source  planValueSourceJSON `json:"source"`
}

type planFormatJSON struct {
	Value  string              `json:"value"`
	Source planValueSourceJSON `json:"source"`
}

type planValueSourceJSON struct {
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

func resolvedFilePairJSON(cwd string, filePair config.ResolvedFilePair) planPairJSON {
	return planPairJSON{
		Plaintext: planPathJSON{
			Path:    filePair.PlaintextPath,
			Display: config.DisplayPath(cwd, filePair.PlaintextPath),
			Source:  valueSourceJSON(filePair.PlaintextSource),
		},
		Encrypted: planPathJSON{
			Path:    filePair.EncryptedPath,
			Display: config.DisplayPath(cwd, filePair.EncryptedPath),
			Source:  valueSourceJSON(filePair.EncryptedSource),
		},
		Format: planFormatJSON{
			Value:  filePair.Format,
			Source: valueSourceJSON(filePair.FormatSource),
		},
		RecipientAliases: filePair.RecipientAliases,
		Recipients:       filePair.Recipients,
		Authorization: planAuthorizationJSON{
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

func valueSourceJSON(source config.ValueSource) planValueSourceJSON {
	return planValueSourceJSON{
		Kind:       source.Kind,
		ConfigPath: source.ConfigPath,
		Detail:     source.Detail,
	}
}
