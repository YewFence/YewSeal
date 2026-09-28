package presentation

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
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
				PlaintextPath: cwd + "/config.toml", EncryptedPath: cwd + "/config.enc.toml", Format: "toml",
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
				PlaintextPath: cwd + "/deploy.env", EncryptedPath: cwd + "/deploy.enc.env", Format: "env",
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
	require.Equal(t, []string{"Plaintext", "Encrypted", "Format", "Aliases"}, strings.Fields(lines[5]))
	require.Equal(t, []string{"config.toml", "config.enc.toml", "toml", "ops,ci"}, strings.Fields(lines[6]))
	require.Equal(t, []string{"deploy.env", "deploy.enc.env", "env", "ops"}, strings.Fields(lines[7]))
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
  plaintext  .yewseal.toml exact
  encrypted  .yewseal.toml exact
  format     .yewseal.toml format
  authz      .yewseal.toml recipients
  registry   ci=.age/ci.txt ops=.age/ops.txt

deploy.env -> deploy.enc.env  format=env  aliases=ops
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
	require.Equal(t, []string{"age1exampleops", "age1exampleci"}, payload.FilePairs[0].Recipients)
	require.Equal(t, "current-directory", payload.FilePairs[0].SelectedBy)
	require.Equal(t, map[string]string{"ops": "/workspace/.age/ops.txt", "ci": "/workspace/.age/ci.txt"}, payload.FilePairs[0].Authorization.RegistrySources)
}

func TestPlanSourcePreservesVerboseConfigListAndEmptySelection(t *testing.T) {
	cfg, selection := planFixture()
	selection.FilePairs = nil
	var out bytes.Buffer
	require.NoError(t, New(&out, nil, true).Plan(cfg, selection, PlanPrintOptions{Source: true, Verbose: true}))
	require.Equal(t, "Loaded 1 config file\nCommand plan\nScope .\nSelected 0 file pairs\n\nConfig files\n1. .yewseal.toml\n\n", out.String())
}
