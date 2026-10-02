package presentation

import (
	"fmt"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestDiagnosticsPaletteIgnoresStdoutNoColor(t *testing.T) {
	terminal := openTestTerminal(t)
	t.Setenv("NO_COLOR", "")
	// Screen skips OSC queries, so COLORFGBG makes both backgrounds deterministic.
	t.Setenv("TERM", "screen-256color")
	previous := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = previous })
	for _, background := range []string{"15;0", "0;15"} {
		t.Run(background, func(t *testing.T) {
			t.Setenv("COLORFGBG", background)
			pal := New(nil, terminal, false).diagnosticsPalette
			for _, role := range []*color.Color{pal.muted, pal.warning, pal.error, pal.success} {
				require.NotNil(t, role)
				require.Contains(t, paint(role, "status"), "\x1b[")
			}
			for _, shared := range []colorPalette{darkPalette, lightPalette} {
				for _, role := range []*color.Color{shared.muted, shared.warning, shared.error, shared.success, shared.format, shared.accent} {
					require.Equal(t, "status", paint(role, "status"))
				}
			}
		})
	}
}

func TestDiagnosticsPaletteTTYOptOut(t *testing.T) {
	terminal := openTestTerminal(t)
	for _, env := range []struct{ noColor, term string }{{"1", "screen-256color"}, {"", "dumb"}} {
		t.Run(env.noColor+env.term, func(t *testing.T) {
			t.Setenv("NO_COLOR", env.noColor)
			t.Setenv("TERM", env.term)
			require.Equal(t, colorPalette{}, diagnosticsPaletteFor(terminal))
		})
	}
}

func openTestTerminal(t *testing.T) *os.File {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, master.Close()) })
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	terminal, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, terminal.Close()) })
	require.True(t, isatty.IsTerminal(terminal.Fd()))
	return terminal
}
