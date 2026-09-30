package presentation

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/rivo/uniseg"
)

// IdentitiesReport 是 identities 命令的呈现模型：解析链的生效来源与
// 被短路来源，加上 registry 反查出的 alias 与未注册警告。
type IdentitiesReport struct {
	Source     string
	Shadowed   []string
	Identities []IdentityEntry
	Warnings   []string
}

// IdentityEntry 是单个身份的呈现行。Secret 总是被填充，但只有
// Reveal 打开时才会进入输出。
type IdentityEntry struct {
	PublicKey string
	Alias     string
	Warning   string
	Secret    string
}

// IdentitiesPrintOptions 控制 identities 报表的输出形态。
type IdentitiesPrintOptions struct {
	JSON   bool
	Reveal bool
}

func (o *Output) Identities(report IdentitiesReport, opts IdentitiesPrintOptions) error {
	for _, warning := range report.Warnings {
		o.Warning(warning)
	}
	for _, entry := range report.Identities {
		if entry.Warning != "" {
			o.statusf(o.diagnosticsPalette.warning, "warning:", " %s (%s)\n", entry.Warning, entry.PublicKey)
		}
	}
	if opts.JSON {
		return printIdentitiesJSON(o, report, opts)
	}
	return printIdentitiesTable(o, report, opts, paletteFor(o.content))
}

// printIdentitiesTable pads plain text before painting (ANSI sequences must
// not affect column width) with the same cell gap the tabwriter layout used.
func printIdentitiesTable(w io.Writer, report IdentitiesReport, opts IdentitiesPrintOptions, pal colorPalette) error {
	if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Source"), report.Source); err != nil {
		return err
	}
	if len(report.Shadowed) > 0 {
		if _, err := fmt.Fprintf(w, "%s %s\n", paint(pal.muted, "Shadowed"), strings.Join(report.Shadowed, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	headers := []string{"Alias", "Public key"}
	if opts.Reveal {
		headers = append(headers, "Secret")
	}
	rows := make([][]string, len(report.Identities))
	for i, entry := range report.Identities {
		row := []string{entry.Alias, entry.PublicKey}
		if opts.Reveal {
			row = append(row, entry.Secret)
		}
		rows[i] = row
	}
	widths := make([]int, len(headers))
	for column, header := range headers {
		widths[column] = uniseg.StringWidth(header)
	}
	for _, row := range rows {
		for column, cell := range row {
			widths[column] = max(widths[column], uniseg.StringWidth(cell))
		}
	}
	columnColors := []*color.Color{pal.accent, nil}
	for range headers[2:] {
		columnColors = append(columnColors, nil)
	}
	var table strings.Builder
	for column, header := range headers {
		table.WriteString(paint(pal.muted, padCell(header, widths[column], column == len(headers)-1)))
	}
	table.WriteString("\n")
	for _, row := range rows {
		for column, cell := range row {
			table.WriteString(paint(columnColors[column], padCell(cell, widths[column], column == len(row)-1)))
		}
		table.WriteString("\n")
	}
	_, err := io.WriteString(w, table.String())
	return err
}

type identitiesJSON struct {
	Command    string              `json:"command"`
	Source     string              `json:"source"`
	Shadowed   []string            `json:"shadowed,omitempty"`
	Identities []identityEntryJSON `json:"identities"`
}

type identityEntryJSON struct {
	PublicKey string `json:"public_key"`
	Alias     string `json:"alias,omitempty"`
	Warning   string `json:"warning,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

func printIdentitiesJSON(w io.Writer, report IdentitiesReport, opts IdentitiesPrintOptions) error {
	payload := identitiesJSON{Command: "identities", Source: report.Source, Shadowed: report.Shadowed, Identities: make([]identityEntryJSON, 0, len(report.Identities))}
	for _, entry := range report.Identities {
		item := identityEntryJSON{PublicKey: entry.PublicKey, Alias: entry.Alias, Warning: entry.Warning}
		if opts.Reveal {
			item.Secret = entry.Secret
		}
		payload.Identities = append(payload.Identities, item)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
