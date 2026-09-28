package presentation

import (
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// colorPalette maps semantic output roles to foreground colors. Dark and
// light terminal backgrounds need different sets: ANSI 16-color slots are
// remapped freely by terminal themes (bright black can render nearly black),
// while the xterm 256-color grays keep fixed RGB values everywhere, so the
// muted role uses a gray from that palette to stay readable across themes.
type colorPalette struct {
	muted    *color.Color // summary keys, table headers, neutral status labels
	format   *color.Color // format tokens
	accent   *color.Color // aliases and provenance paths
	warning  *color.Color // warning status labels
	error     *color.Color // failure status labels
	success  *color.Color // completed-action status labels
	diffHead *color.Color // ---/+++ headers
	diffHunk *color.Color // @@ hunks
	diffDel  *color.Color // removed lines
	diffIns  *color.Color // added lines
}

var (
	darkPalette = colorPalette{
		muted:    fg256(245),
		format:   color.New(color.FgHiCyan),
		accent:   color.New(color.FgHiYellow),
		warning:  color.New(color.FgHiYellow),
		error:     color.New(color.FgHiRed),
		success:  color.New(color.FgHiGreen),
		diffHead: color.New(color.FgHiCyan, color.Bold),
		diffHunk: color.New(color.FgHiMagenta),
		diffDel:  color.New(color.FgHiRed),
		diffIns:  color.New(color.FgHiGreen),
	}
	lightPalette = colorPalette{
		muted:    fg256(242),
		format:   color.New(color.FgCyan),
		accent:   color.New(color.FgYellow),
		warning:  color.New(color.FgYellow),
		error:     color.New(color.FgRed),
		success:  color.New(color.FgGreen),
		diffHead: color.New(color.FgCyan, color.Bold),
		diffHunk: color.New(color.FgMagenta),
		diffDel:  color.New(color.FgRed),
		diffIns:  color.New(color.FgGreen),
	}
)

// fg256 builds a color from the xterm 256-color palette.
func fg256(n int) *color.Color {
	return color.New(color.Attribute(38), color.Attribute(5), color.Attribute(n))
}

// paint colors s with c; unset roles and empty strings stay untouched.
func paint(c *color.Color, s string) string {
	if c == nil || s == "" {
		return s
	}
	return c.Sprint(s)
}

var (
	backgroundOnce   sync.Once
	backgroundIsDark bool
)

// terminalHasDarkBackground asks the terminal for its background color (OSC
// 11) once per process and falls back on terminals that cannot answer.
func terminalHasDarkBackground() bool {
	backgroundOnce.Do(func() { backgroundIsDark = termenv.HasDarkBackground() })
	return backgroundIsDark
}

// paletteFor picks the palette for w: no colors unless w is the process
// stdout with colors enabled (fatih/color honors NO_COLOR and non-TTY
// stdout), then the terminal background chooses dark or light.
func paletteFor(w io.Writer) colorPalette {
	if w != os.Stdout || color.NoColor {
		return colorPalette{}
	}
	if terminalHasDarkBackground() {
		return darkPalette
	}
	return lightPalette
}

// diagnosticsPaletteFor picks the palette for the diagnostic stream. It
// checks NO_COLOR, TERM=dumb, and the writer's own TTY state, because
// fatih/color's global NoColor only reflects the stdout stream: stderr may
// be redirected while stdout is an interactive terminal, and vice versa.
func diagnosticsPaletteFor(w io.Writer) colorPalette {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return colorPalette{}
	}
	// termenv keeps its TTY probe private, so an isatty check on the writer's
	// file descriptor decides whether colors can go out at all.
	file, ok := w.(*os.File)
	if !ok || !isatty.IsTerminal(file.Fd()) {
		return colorPalette{}
	}
	if termenv.NewOutput(w).HasDarkBackground() {
		return darkPalette
	}
	return lightPalette
}

// forcedDiffPalette colors diff output even when stdout is redirected, for
// --color always. It forces only the diff roles in place, which nothing but
// diff rendering ever touches; the plan roles of the shared palettes keep
// honoring NO_COLOR and non-TTY stdout.
func forcedDiffPalette() colorPalette {
	base := lightPalette
	if terminalHasDarkBackground() {
		base = darkPalette
	}
	for _, c := range []*color.Color{base.diffHead, base.diffHunk, base.diffDel, base.diffIns} {
		c.EnableColor()
	}
	return base
}
