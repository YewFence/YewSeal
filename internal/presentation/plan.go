package presentation

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/fatih/color"
	"github.com/rivo/uniseg"
)

type PlanPrintOptions struct {
	JSON    bool
	Verbose bool
	Source  bool
}

func (o *Output) Plan(cfg *config.Config, selection config.ResolvedSelection, opts PlanPrintOptions) error {
	if opts.JSON {
		return printPlanJSON(o, cfg, selection)
	}
	return printPlanText(o, cfg, selection, opts, paletteFor(o.content))
}

func printPlanText(w io.Writer, cfg *config.Config, selection config.ResolvedSelection, opts PlanPrintOptions, pal colorPalette) error {
	cwd := config.CurrentDir(cfg)
	scope := config.DisplayPath(cwd, selection.CurrentDirScope)
	if scope == "" {
		scope = "."
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Loaded"), countNoun(len(selection.ConfigFiles), "config file")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Command"), selection.Command); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Scope"), scope); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Selected"), countNoun(len(selection.FilePairs), "file pair")); err != nil {
		return err
	}
	if opts.Verbose && len(selection.ConfigFiles) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, paint(pal.muted, "Config files")); err != nil {
			return err
		}
		for i, file := range selection.ConfigFiles {
			if _, err := fmt.Fprintf(w, "%d. %s\n", i+1, config.DisplayPath(cwd, file.Path)); err != nil {
				return err
			}
		}
	}

	rules := cfg.GetExcludeRules()
	if len(rules) > 0 {
		if _, err := fmt.Fprintln(w, "\nExclude rules (loaded configuration)"); err != nil {
			return err
		}
		for _, rule := range rules {
			if _, err := fmt.Fprintf(w, "  %s exclude[%d] = %q\n", config.DisplayPath(cwd, rule.ConfigPath), rule.Index, rule.Pattern); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if opts.Source {
		return printPlanSources(w, cwd, selection.FilePairs, pal)
	}
	return printPlanTable(w, cwd, selection.FilePairs, pal)
}

// printPlanTable renders the mapping table. Padding is computed on the
// plain text and colors are applied afterwards, because tabwriter counts
// ANSI escape sequences as cell width and would misalign colored columns.
func printPlanTable(w io.Writer, cwd string, filePairs []config.ResolvedFilePair, pal colorPalette) error {
	headers := []string{"Plaintext", "Encrypted", "Format", "Aliases", "PlaintextMode"}
	rows := make([][]tableCell, len(filePairs))
	for i, filePair := range filePairs {
		rows[i] = []tableCell{
			{text: config.DisplayPath(cwd, filePair.PlaintextPath), suffix: diskMarker(filePair.PlaintextExists, true), suffixColor: pal.warning},
			{text: config.DisplayPath(cwd, filePair.EncryptedPath), suffix: diskMarker(filePair.EncryptedExists, false), suffixColor: pal.error},
			{text: filePair.Format, color: pal.format},
			{text: strings.Join(filePair.RecipientAliases, ","), color: pal.accent},
			{text: filePair.PlaintextMode},
		}
	}
	widths := make([]int, len(headers))
	for column, header := range headers {
		widths[column] = uniseg.StringWidth(header)
	}
	for _, row := range rows {
		for column, cell := range row {
			widths[column] = max(widths[column], cell.width())
		}
	}
	var table strings.Builder
	for column, header := range headers {
		table.WriteString(paint(pal.muted, padCell(header, widths[column], column == len(headers)-1)))
	}
	table.WriteString("\n")
	for _, row := range rows {
		for column, cell := range row {
			table.WriteString(cell.render(widths[column], column == len(row)-1))
		}
		table.WriteString("\n")
	}
	_, err := io.WriteString(w, table.String())
	return err
}

// tableCell 是表格的一个单元格：text 着 color 色，suffix 是磁盘状态后缀，
// 单独着 suffixColor，宽度按纯文本计，与 padCell 的对齐规则一致。
type tableCell struct {
	text        string
	suffix      string
	color       *color.Color
	suffixColor *color.Color
}

func (c tableCell) width() int {
	width := uniseg.StringWidth(c.text)
	if c.suffix != "" {
		width += 1 + uniseg.StringWidth(c.suffix)
	}
	return width
}

func (c tableCell) render(width int, final bool) string {
	cell := paint(c.color, c.text)
	if c.suffix != "" {
		cell += " " + paint(c.suffixColor, c.suffix)
	}
	if final {
		return cell
	}
	return cell + strings.Repeat(" ", width-c.width()+2)
}

// diskMarker returns the asymmetric disk-state marker for a path: a
// present plaintext is noteworthy (local work copy) and a missing ciphertext
// is an anomaly, while an absent plaintext is the clean-checkout default and
// carries no marker.
func diskMarker(exists *bool, plaintextColumn bool) string {
	if exists == nil {
		return ""
	}
	if *exists {
		if plaintextColumn {
			return "(existing)"
		}
		return ""
	}
	if plaintextColumn {
		return ""
	}
	return "(missing)"
}

// describePath renders a muted path with its colored disk-state marker for
// the describe layout.
func describePath(path string, exists *bool, plaintextColumn bool, pal colorPalette) string {
	return paint(pal.muted, path) + diskMarkerSuffix(exists, plaintextColumn, pal)
}

func diskMarkerSuffix(exists *bool, plaintextColumn bool, pal colorPalette) string {
	marker := diskMarker(exists, plaintextColumn)
	if marker == "" {
		return ""
	}
	colorRole := pal.error
	if plaintextColumn {
		colorRole = pal.warning
	}
	return " " + paint(colorRole, marker)
}

// padCell pads a table cell to its display width plus the two-space gap,
// except after the final column.
func padCell(cell string, width int, final bool) string {
	if final {
		return cell
	}
	return cell + strings.Repeat(" ", width-uniseg.StringWidth(cell)+2)
}

func printPlanSources(w io.Writer, cwd string, filePairs []config.ResolvedFilePair, pal colorPalette) error {
	for i, filePair := range filePairs {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%s %s %s  %s%s  %s%s\n",
			describePath(config.DisplayPath(cwd, filePair.PlaintextPath), filePair.PlaintextExists, true, pal),
			paint(pal.muted, "->"),
			describePath(config.DisplayPath(cwd, filePair.EncryptedPath), filePair.EncryptedExists, false, pal),
			paint(pal.muted, "format="),
			paint(pal.format, filePair.Format),
			paint(pal.muted, "aliases="),
			paint(pal.accent, strings.Join(filePair.RecipientAliases, ","))); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  plaintext_mode %s\n", filePair.PlaintextMode); err != nil {
			return err
		}
		if marker := diskMarkerSummary(filePair); marker != "" {
			if _, err := fmt.Fprintf(w, "  on disk     %s\n", marker); err != nil {
				return err
			}
		}
		for _, field := range []struct{ label, source string }{
			{"plaintext", formatValueSource(filePair.PlaintextSource, cwd, pal)},
			{"encrypted", formatValueSource(filePair.EncryptedSource, cwd, pal)},
			{"format", formatValueSource(filePair.FormatSource, cwd, pal)},
			{"authz", formatAuthorizationSource(filePair, cwd, pal)},
			{"registry", formatRegistrySources(filePair.RecipientInfo, cwd, pal)},
		} {
			if _, err := fmt.Fprintf(w, "  %s%s\n", paint(pal.muted, fmt.Sprintf("%-11s", field.label)), field.source); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatValueSource(source config.ValueSource, cwd string, pal colorPalette) string {
	if source.ConfigPath == "" {
		return config.FormatValueSource(source, cwd)
	}
	detail := source.Detail
	if detail == "" {
		detail = config.FormatValueSource(config.ValueSource{Kind: source.Kind}, cwd)
	}
	return paint(pal.accent, config.DisplayPath(cwd, source.ConfigPath)) + " " + paint(pal.muted, detail)
}

func printPlanJSON(w io.Writer, cfg *config.Config, selection config.ResolvedSelection) error {
	cwd := config.CurrentDir(cfg)
	payload := planJSON{
		Command: selection.Command,
		Scope:   config.DisplayPath(cwd, selection.CurrentDirScope),
		Exclude: make([]planExcludeJSON, 0),
	}
	for _, file := range selection.ConfigFiles {
		payload.ConfigFiles = append(payload.ConfigFiles, file.Path)
	}
	for _, filePair := range selection.FilePairs {
		payload.FilePairs = append(payload.FilePairs, resolvedFilePairJSON(cwd, filePair))
	}
	for _, rule := range cfg.GetExcludeRules() {
		payload.Exclude = append(payload.Exclude, planExcludeJSON{
			Pattern: rule.Pattern, Config: rule.ConfigPath, Index: rule.Index,
		})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

type planJSON struct {
	Command     string            `json:"command"`
	Scope       string            `json:"scope,omitempty"`
	ConfigFiles []string          `json:"config_files"`
	FilePairs   []planPairJSON    `json:"file_pairs"`
	Exclude     []planExcludeJSON `json:"exclude"`
}

type planExcludeJSON struct {
	Pattern string `json:"pattern"`
	Config  string `json:"config"`
	Index   int    `json:"index"`
}

type planPairJSON struct {
	PlaintextMode    string                `json:"plaintext_mode"`
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
	Exists  *bool               `json:"exists,omitempty"`
	Source  planValueSourceJSON `json:"source"`
}

// diskMarkerSummary describes both sides' disk state in the describe layout:
// the clean-checkout default (ciphertext only) reads as "encrypted only".
func diskMarkerSummary(filePair config.ResolvedFilePair) string {
	if filePair.PlaintextExists == nil || filePair.EncryptedExists == nil {
		return ""
	}
	switch {
	case *filePair.PlaintextExists && *filePair.EncryptedExists:
		return "both plaintext and encrypted"
	case *filePair.PlaintextExists:
		return "plaintext only (not yet encrypted)"
	case *filePair.EncryptedExists:
		return "encrypted only"
	default:
		return "neither on disk"
	}
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

func formatAuthorizationSource(filePair config.ResolvedFilePair, cwd string, pal colorPalette) string {
	if filePair.RecipientInfo.EffectiveSource.Kind != "" {
		return formatValueSource(filePair.RecipientInfo.EffectiveSource, cwd, pal)
	}
	return filePair.RecipientInfo.Kind
}

func formatRegistrySources(info config.RecipientProvenance, cwd string, pal colorPalette) string {
	aliases := make([]string, 0, len(info.RegistrySources))
	for alias := range info.RegistrySources {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	formatted := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		formatted = append(formatted, alias+"="+paint(pal.accent, config.DisplayPath(cwd, info.RegistrySources[alias])))
	}
	return strings.Join(formatted, " ")
}

func resolvedFilePairJSON(cwd string, filePair config.ResolvedFilePair) planPairJSON {
	return planPairJSON{
		PlaintextMode: filePair.PlaintextMode,
		Plaintext: planPathJSON{
			Path:    filePair.PlaintextPath,
			Display: config.DisplayPath(cwd, filePair.PlaintextPath),
			Exists:  filePair.PlaintextExists,
			Source:  valueSourceJSON(filePair.PlaintextSource),
		},
		Encrypted: planPathJSON{
			Path:    filePair.EncryptedPath,
			Display: config.DisplayPath(cwd, filePair.EncryptedPath),
			Exists:  filePair.EncryptedExists,
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
