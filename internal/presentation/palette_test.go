package presentation

import (
	"bytes"
	"regexp"
	"testing"

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
