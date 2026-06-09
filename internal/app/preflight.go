package app

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/task"
)

type PreflightResult struct {
	Selection     config.ResolvedSelection
	KeyFile       string
	PublicKey     string
	Parallel      int
	Force         bool
	Verbose       bool
	MetadataPairs []config.ResolvedFilePair
	MetadataScope string
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
		FormatRules:          req.FormatRules,
		UnknownAsBinary:      req.UnknownAsBinary,
		UnknownAsBinarySet:   req.UnknownAsBinarySet,
		AllowEmptyTarget:     true,
		UseConfiguredDefault: true,
	})
	if err != nil {
		return PreflightResult{}, err
	}

	publicKeyCandidate := req.PublicKey
	if publicKeyCandidate == "" {
		publicKeyCandidate = cfg.GetPublicKey()
	}
	resolvedPublicKey, err := agekey.GetPublicKey(publicKeyCandidate, req.KeyFile, req.Verbose)
	if err != nil {
		return PreflightResult{}, err
	}

	metadataPairs, metadataScope := metadataPairsForSelection(selection)
	return PreflightResult{
		Selection:     selection,
		KeyFile:       req.KeyFile,
		PublicKey:     resolvedPublicKey,
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
		FormatRules:          req.FormatRules,
		UnknownAsBinary:      req.UnknownAsBinary,
		UnknownAsBinarySet:   req.UnknownAsBinarySet,
		AllowEmptyTarget:     true,
		UseConfiguredDefault: true,
	})
	if err != nil {
		return PreflightResult{}, err
	}
	if _, err := agekey.GetAgeKey(req.KeyFile); err != nil {
		return PreflightResult{}, err
	}

	metadataPairs, metadataScope := metadataPairsForSelection(selection)
	return PreflightResult{
		Selection:     selection,
		KeyFile:       req.KeyFile,
		Parallel:      req.Parallel,
		Force:         req.Force,
		Verbose:       req.Verbose,
		MetadataPairs: metadataPairs,
		MetadataScope: metadataScope,
	}, nil
}

func PrintEncryptPreflight(w io.Writer, cfg *config.Config, req EncryptRequest, opts PreflightPrintOptions) error {
	result, err := PreflightEncrypt(cfg, req)
	if err != nil {
		return err
	}
	return PrintPreflight(w, cfg, result, opts)
}

func PrintDecryptPreflight(w io.Writer, cfg *config.Config, req DecryptRequest, opts PreflightPrintOptions) error {
	result, err := PreflightDecrypt(cfg, req)
	if err != nil {
		return err
	}
	return PrintPreflight(w, cfg, result, opts)
}

type PlanRequest struct {
	Output             string
	OutputSet          bool
	Format             string
	Target             string
	Patterns           []string
	FormatRules        []string
	UnknownAsBinary    bool
	UnknownAsBinarySet bool
	Parallel           int
	Verbose            bool
}

func PrintPlan(w io.Writer, cfg *config.Config, req PlanRequest, opts PreflightPrintOptions) error {
	selection, err := config.ResolvePlanSelection(cfg, config.SelectionOptions{
		Target:             req.Target,
		Output:             req.Output,
		OutputSet:          req.OutputSet,
		Format:             req.Format,
		Patterns:           req.Patterns,
		FormatRules:        req.FormatRules,
		UnknownAsBinary:    req.UnknownAsBinary,
		UnknownAsBinarySet: req.UnknownAsBinarySet,
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
	if _, err := fmt.Fprintln(tw, "Plaintext\tP.Source\tEncrypted\tE.Source\tFormat\tF.Source\tSelected By"); err != nil {
		return err
	}
	for _, filePair := range selection.FilePairs {
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			config.DisplayPath(cwd, filePair.PlaintextPath),
			config.FormatValueSource(filePair.PlaintextSource, cwd),
			config.DisplayPath(cwd, filePair.EncryptedPath),
			config.FormatValueSource(filePair.EncryptedSource, cwd),
			filePair.Format,
			config.FormatValueSource(filePair.FormatSource, cwd),
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
	Plaintext  preflightPathJSON   `json:"plaintext"`
	Encrypted  preflightPathJSON   `json:"encrypted"`
	Format     preflightFormatJSON `json:"format"`
	SelectedBy string              `json:"selected_by"`
	Source     string              `json:"source"`
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
		SelectedBy: filePair.SelectedBy,
		Source:     filePair.Source,
	}
}

func valueSourceJSON(source config.ValueSource) preflightValueSourceJSON {
	return preflightValueSourceJSON{
		Kind:       source.Kind,
		ConfigPath: source.ConfigPath,
		Detail:     source.Detail,
	}
}
