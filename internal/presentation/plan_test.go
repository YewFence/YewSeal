package presentation

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/rivo/uniseg"
	"github.com/stretchr/testify/require"
)

func planFixture() (*config.Config, config.ResolvedSelection) {
	const cwd = "/workspace"
	configPath := cwd + "/.yewseal.toml"
	return &config.Config{CurrentDir: cwd}, config.ResolvedSelection{
		Command: "plan", CurrentDirScope: cwd,
		ConfigFiles: []config.LoadedFile{{Path: configPath}},
		FilePairs: []config.ResolvedFilePair{
			{
				PlaintextPath: cwd + "/config.toml", EncryptedPath: cwd + "/config.enc.toml", Format: "toml", PlaintextMode: "delivery",
				PlaintextSource:  config.ValueSource{Kind: "exact", ConfigPath: configPath},
				EncryptedSource:  config.ValueSource{Kind: "exact", ConfigPath: configPath},
				FormatSource:     config.ValueSource{Kind: "config-format", ConfigPath: configPath, Detail: "format"},
				RecipientAliases: []string{"ops", "ci"}, Recipients: []string{"age1exampleops", "age1exampleci"},
				RecipientInfo: config.RecipientProvenance{
					Kind: "file", EffectiveSource: config.ValueSource{Kind: "file", ConfigPath: configPath, Detail: "recipients"},
					RegistrySources: map[string]string{"ops": cwd + "/.age/ops.txt", "ci": cwd + "/.age/ci.txt"},
				},
				SelectedBy: "current-directory", Source: "exact",
			},
			{
				PlaintextPath: cwd + "/deploy.env", EncryptedPath: cwd + "/deploy.enc.env", Format: "env", PlaintextMode: "inplace",
				PlaintextSource:  config.ValueSource{Kind: "scan"},
				EncryptedSource:  config.ValueSource{Kind: "protocol"},
				FormatSource:     config.ValueSource{Kind: "filename"},
				RecipientAliases: []string{"ops"}, Recipients: []string{"age1exampleops"},
				RecipientInfo: config.RecipientProvenance{
					Kind: "defaults", EffectiveSource: config.ValueSource{Kind: "defaults", ConfigPath: configPath, Detail: "recipients.defaults"},
					RegistrySources: map[string]string{"ops": cwd + "/.age/ops.txt"},
				},
				SelectedBy: "current-directory", Source: "scan",
			},
		},
	}
}

func TestPlanDefaultTableHasOnlyCoreColumns(t *testing.T) {
	cfg, selection := planFixture()
	var out bytes.Buffer
	require.NoError(t, New(&out, nil, false).Plan(cfg, selection, PlanPrintOptions{}))
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	require.Equal(t, []string{"Loaded 1 config file", "Command plan", "Scope .", "Selected 2 file pairs", ""}, lines[:5])
	for _, line := range lines[5:] {
		require.LessOrEqual(t, len(line), 80)
	}
	require.Equal(t, []string{"Plaintext", "Encrypted", "Format", "Aliases", "PlaintextMode"}, strings.Fields(lines[5]))
	require.Equal(t, []string{"config.toml", "config.enc.toml", "toml", "ops,ci", "delivery"}, strings.Fields(lines[6]))
	require.Equal(t, []string{"deploy.env", "deploy.enc.env", "env", "ops", "inplace"}, strings.Fields(lines[7]))
	require.Len(t, lines, 8)
	require.NotContains(t, out.String(), "age1example")
	require.NotContains(t, out.String(), "Selected By")
	require.NotContains(t, out.String(), ".Source")
}

func TestPlanSourceDescribeShowsFieldOrigins(t *testing.T) {
	cfg, selection := planFixture()
	var out bytes.Buffer
	require.NoError(t, New(&out, nil, false).Plan(cfg, selection, PlanPrintOptions{Source: true}))
	require.Equal(t, `Loaded 1 config file
Command plan
Scope .
Selected 2 file pairs

config.toml -> config.enc.toml  format=toml  aliases=ops,ci
  plaintext_mode delivery
  plaintext  .yewseal.toml exact
  encrypted  .yewseal.toml exact
  format     .yewseal.toml format
  authz      .yewseal.toml recipients
  registry   ci=.age/ci.txt ops=.age/ops.txt

deploy.env -> deploy.enc.env  format=env  aliases=ops
  plaintext_mode inplace
  plaintext  scan
  encrypted  protocol
  format     filename
  authz      .yewseal.toml recipients.defaults
  registry   ops=.age/ops.txt
`, out.String())
	require.NotContains(t, out.String(), "age1example")
	require.NotContains(t, out.String(), "current-directory")
}

func TestPlanJSONIgnoresSourceAndVerbose(t *testing.T) {
	cfg, selection := planFixture()
	var baseline, combined bytes.Buffer
	require.NoError(t, New(&baseline, nil, false).Plan(cfg, selection, PlanPrintOptions{JSON: true}))
	require.NoError(t, New(&combined, nil, true).Plan(cfg, selection, PlanPrintOptions{JSON: true, Source: true, Verbose: true}))
	require.Equal(t, baseline.String(), combined.String())
	var payload struct {
		Command     string   `json:"command"`
		ConfigFiles []string `json:"config_files"`
		FilePairs   []struct {
			PlaintextMode string   `json:"plaintext_mode"`
			Recipients    []string `json:"recipients"`
			SelectedBy    string   `json:"selected_by"`
			Authorization struct {
				RegistrySources map[string]string `json:"registry_sources"`
			} `json:"authorization"`
		} `json:"file_pairs"`
	}
	require.NoError(t, json.Unmarshal(combined.Bytes(), &payload))
	require.Equal(t, "plan", payload.Command)
	require.Equal(t, []string{"/workspace/.yewseal.toml"}, payload.ConfigFiles)
	require.Len(t, payload.FilePairs, 2)
	require.Equal(t, "delivery", payload.FilePairs[0].PlaintextMode)
	require.Equal(t, "inplace", payload.FilePairs[1].PlaintextMode)
	require.Equal(t, []string{"age1exampleops", "age1exampleci"}, payload.FilePairs[0].Recipients)
	require.Equal(t, "current-directory", payload.FilePairs[0].SelectedBy)
	require.Equal(t, map[string]string{"ops": "/workspace/.age/ops.txt", "ci": "/workspace/.age/ci.txt"}, payload.FilePairs[0].Authorization.RegistrySources)
}

func TestPlanSourcePreservesPaths(t *testing.T) {
	withColorsEnabled(t)
	for _, path := range []string{"team keys/.yewseal.toml", "dir=prod/.yewseal.toml", "team keys/dir=prod/.yewseal.toml"} {
		t.Run(path, func(t *testing.T) {
			cfg, selection := planFixture()
			pair := &selection.FilePairs[0]
			pair.PlaintextSource.ConfigPath = "/workspace/" + path
			pair.EncryptedSource.ConfigPath = "/workspace/" + path
			pair.FormatSource.ConfigPath = "/workspace/" + path
			pair.RecipientInfo.EffectiveSource.ConfigPath = "/workspace/" + path
			pair.RecipientInfo.RegistrySources = map[string]string{"ops": "/workspace/" + path, "ci": "/workspace/" + path}
			selection.FilePairs = selection.FilePairs[:1]
			for _, pal := range []colorPalette{{}, darkPalette, lightPalette} {
				var out bytes.Buffer
				require.NoError(t, printPlanText(&out, cfg, selection, PlanPrintOptions{Source: true}, pal))
				plain := ansiEscapes.ReplaceAllString(out.String(), "")
				require.Contains(t, plain, "  plaintext  "+path+" exact\n")
				require.Contains(t, plain, "  encrypted  "+path+" exact\n")
				require.Contains(t, plain, "  format     "+path+" format\n")
				require.Contains(t, plain, "  authz      "+path+" recipients\n")
				require.Contains(t, plain, "  registry   ci="+path+" ops="+path+"\n")
				require.Contains(t, out.String(), paint(pal.accent, path))
			}
		})
	}
}

func TestPlanTableAlignsDisplayColumns(t *testing.T) {
	withColorsEnabled(t)
	for _, path := range []string{"\u914d\u7f6e.toml", "\u00e9.toml", "e\u0301.toml", "\U0001f469\u200d\U0001f4bb.toml"} {
		t.Run(path, func(t *testing.T) {
			cfg, selection := planFixture()
			selection.FilePairs[0].PlaintextPath = "/workspace/" + path
			selection.FilePairs[0].EncryptedPath = "/workspace/" + path + ".enc"
			for _, pal := range []colorPalette{{}, darkPalette, lightPalette} {
				var out bytes.Buffer
				require.NoError(t, printPlanTable(&out, cfg.CurrentDir, selection.FilePairs, pal))
				lines := strings.Split(strings.TrimSuffix(ansiEscapes.ReplaceAllString(out.String(), ""), "\n"), "\n")
				for column := 1; column < 5; column++ {
					positions := make([]int, len(lines))
					for i, line := range lines {
						cell := strings.Fields(line)[column]
						positions[i] = uniseg.StringWidth(line[:strings.LastIndex(line, cell)])
					}
					require.Equal(t, positions[0], positions[1], "column %d", column)
					require.Equal(t, positions[0], positions[2], "column %d", column)
				}
			}
		})
	}
}

func TestPadCellUsesDisplayWidth(t *testing.T) {
	for _, cell := range []string{"ascii", "\u914d\u7f6e", "\u00e9", "e\u0301", "\U0001f469\u200d\U0001f4bb"} {
		t.Run(cell, func(t *testing.T) {
			require.Equal(t, 12, uniseg.StringWidth(padCell(cell, 10, false)))
			require.Equal(t, cell, padCell(cell, 10, true))
		})
	}
}

func TestPlanSourcePreservesVerboseConfigListAndEmptySelection(t *testing.T) {
	cfg, selection := planFixture()
	selection.FilePairs = nil
	var out bytes.Buffer
	require.NoError(t, New(&out, nil, true).Plan(cfg, selection, PlanPrintOptions{Source: true, Verbose: true}))
	require.Equal(t, "Loaded 1 config file\nCommand plan\nScope .\nSelected 0 file pairs\n\nConfig files\n1. .yewseal.toml\n\n", out.String())
}
