package presentation

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"testing"

	"github.com/YewFence/YewSeal/internal/task"
	"github.com/YewFence/YewSeal/internal/verify"
	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

func TestPlanColorizedRenderMatchesPlainText(t *testing.T) {
	cfg, selection := planFixture()
	var plain, colored bytes.Buffer
	require.NoError(t, New(&plain, nil, false).Plan(cfg, selection, PlanPrintOptions{Source: true}))
	previous := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previous })
	require.NoError(t, printPlanText(&colored, cfg, selection, PlanPrintOptions{Source: true}, darkPalette))
	require.Contains(t, colored.String(), "\x1b[38;5;245m") // muted fixed gray
	require.Contains(t, colored.String(), "\x1b[96m")       // format bright cyan
	require.Contains(t, colored.String(), "\x1b[93m")       // accent bright yellow
	require.Equal(t, plain.String(), ansiEscapes.ReplaceAllString(colored.String(), ""))
}

func TestPlanTableColorizedRenderMatchesPlainText(t *testing.T) {
	cfg, selection := planFixture()
	var plain, colored bytes.Buffer
	require.NoError(t, New(&plain, nil, false).Plan(cfg, selection, PlanPrintOptions{}))
	previous := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previous })
	require.NoError(t, printPlanText(&colored, cfg, selection, PlanPrintOptions{}, lightPalette))
	require.Contains(t, colored.String(), "\x1b[36m") // format cyan
	require.Contains(t, colored.String(), "\x1b[33m") // accent yellow
	require.Equal(t, plain.String(), ansiEscapes.ReplaceAllString(colored.String(), ""))
}

var ansiEscapes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func withColorsEnabled(t *testing.T) {
	t.Helper()
	previous := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previous })
}

func TestDiagnosticStatusColorsMatchPlainOutput(t *testing.T) {
	render := func(pal colorPalette) string {
		var diag bytes.Buffer
		out := New(io.Discard, &diag, false)
		out.diagnosticsPalette = pal
		out.Warning("recipient missing")
		out.Error(errors.New("boom"))
		out.FileCompleted(task.Result{Status: task.Failed, SourceFile: "config.toml", Error: errors.New("nope")})
		out.FileCompleted(task.Result{Status: task.Skipped, SourceFile: "gone.toml", Outcome: task.OutcomeMissingPlaintext})
		out.Edited("config.toml", true)
		out.Edited("other.toml", false)
		return diag.String()
	}
	plain := render(colorPalette{})
	withColorsEnabled(t)
	colored := render(darkPalette)
	require.Contains(t, colored, "\x1b[93mWARNING")     // warning bright yellow
	require.Contains(t, colored, "\x1b[91mFAILED")      // error bright red
	require.Contains(t, colored, "\x1b[91mError:")
	require.Contains(t, colored, "\x1b[38;5;245mSKIPPED") // muted fixed gray
	require.Contains(t, colored, "\x1b[92mUpdated")     // success bright green
	require.Equal(t, plain, ansiEscapes.ReplaceAllString(colored, ""))
}

func TestVerifyReportColorizedMatchesPlain(t *testing.T) {
	report := &verify.Report{
		Findings: []verify.Finding{
			{Severity: verify.SeverityError, Code: "e1", PlaintextPath: "a.toml", EncryptedPath: "a.enc.toml", Message: "broken", Hint: "rotate"},
			{Severity: verify.SeverityWarning, Code: "w1", Message: "drift"},
		},
		SkipReasons: []string{"no identities"},
	}
	identity := func(s string) string { return s }
	var plain, colored bytes.Buffer
	require.NoError(t, printVerifyReport(&plain, report, colorPalette{}, identity))
	withColorsEnabled(t)
	require.NoError(t, printVerifyReport(&colored, report, darkPalette, identity))
	require.Contains(t, colored.String(), "\x1b[91mERROR")
	require.Contains(t, colored.String(), "\x1b[93mWARNING")
	require.Contains(t, colored.String(), "\x1b[38;5;245m  hint:")
	require.Equal(t, plain.String(), ansiEscapes.ReplaceAllString(colored.String(), ""))
}

func TestIdentitiesTableColorizedMatchesPlain(t *testing.T) {
	report := IdentitiesReport{
		Source:   ".age/keys.txt",
		Shadowed: []string{"env:YEWSEAL_AGE_IDENTITIES"},
		Identities: []IdentityEntry{{Alias: "owner", PublicKey: "age1abc"}, {PublicKey: "age1def", Secret: "AGE-SECRET-KEY-1"}},
	}
	var plain, colored bytes.Buffer
	require.NoError(t, printIdentitiesTable(&plain, report, IdentitiesPrintOptions{Reveal: true}, colorPalette{}))
	withColorsEnabled(t)
	require.NoError(t, printIdentitiesTable(&colored, report, IdentitiesPrintOptions{Reveal: true}, darkPalette))
	require.Contains(t, colored.String(), "\x1b[38;5;245mSource")
	require.Contains(t, colored.String(), "\x1b[93mowner")
	require.Equal(t, plain.String(), ansiEscapes.ReplaceAllString(colored.String(), ""))
}

func TestDiagnosticsPaletteForNonTTYOrNoColorIsPlain(t *testing.T) {
	requireZeroPalette(t, diagnosticsPaletteFor(&bytes.Buffer{}))
	t.Setenv("NO_COLOR", "1")
	requireZeroPalette(t, diagnosticsPaletteFor(io.Discard))
}

func TestSeverityColorMapping(t *testing.T) {
	pal := darkPalette
	require.Same(t, pal.error, severityColor(pal, verify.SeverityError))
	require.Same(t, pal.warning, severityColor(pal, verify.SeverityWarning))
	require.Same(t, pal.muted, severityColor(pal, verify.Severity("other")))
}

func TestPaletteForNonStdoutWriterIsPlain(t *testing.T) {
	requireZeroPalette(t, paletteFor(&bytes.Buffer{}))
	previous := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = previous })
	requireZeroPalette(t, paletteFor(bytes.NewBuffer(nil)))
}

func TestDiffPaletteForcesOnlyDiffRoles(t *testing.T) {
	forced := forcedDiffPalette()
	previous := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = previous })
	require.Contains(t, forced.diffHead.Sprint("x"), "\x1b[") // forced diff roles ignore NoColor
	require.Equal(t, "x", lightPalette.muted.Sprint("x")) // plan roles stay respectful
	require.Equal(t, "x", lightPalette.format.Sprint("x"))
	require.Equal(t, "x", lightPalette.accent.Sprint("x"))
}

func requireZeroPalette(t *testing.T, pal colorPalette) {
	t.Helper()
	require.Nil(t, pal.muted)
	require.Nil(t, pal.format)
	require.Nil(t, pal.accent)
	require.Nil(t, pal.diffHead)
}
