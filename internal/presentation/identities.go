package presentation

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
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
			o.diagnostic(fmt.Sprintf("warning: %s (%s)\n", entry.Warning, entry.PublicKey))
		}
	}
	if opts.JSON {
		return printIdentitiesJSON(o, report, opts)
	}
	return printIdentitiesTable(o, report, opts)
}

func printIdentitiesTable(w io.Writer, report IdentitiesReport, opts IdentitiesPrintOptions) error {
	if _, err := fmt.Fprintf(w, "Source %s\n", report.Source); err != nil {
		return err
	}
	if len(report.Shadowed) > 0 {
		if _, err := fmt.Fprintf(w, "Shadowed %s\n", strings.Join(report.Shadowed, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := "Alias\tPublic key"
	if opts.Reveal {
		header += "\tSecret"
	}
	if _, err := fmt.Fprintln(tw, header); err != nil {
		return err
	}
	for _, entry := range report.Identities {
		row := entry.Alias + "\t" + entry.PublicKey
		if opts.Reveal {
			row += "\t" + entry.Secret
		}
		if _, err := fmt.Fprintln(tw, row); err != nil {
			return err
		}
	}
	return tw.Flush()
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
	payload := identitiesJSON{Command: "identities", Source: report.Source, Shadowed: report.Shadowed}
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
