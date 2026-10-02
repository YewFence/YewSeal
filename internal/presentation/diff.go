package presentation

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

func ResolveDiffColor(mode string, w io.Writer) (bool, error) {
	switch mode {
	case "", "auto":
		return w == os.Stdout && !color.NoColor, nil
	case "always":
		return true, nil
	case "never":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported color mode %q (supported: auto, always, never)", mode)
	}
}

func (o *Output) Diff(text string, colorEnabled bool) error {
	pal := colorPalette{}
	if colorEnabled {
		pal = forcedDiffPalette()
	}
	_, err := o.Write([]byte(highlightUnifiedDiff(text, pal)))
	return err
}

func highlightUnifiedDiff(text string, pal colorPalette) string {
	if text == "" || pal.diffHead == nil {
		return text
	}
	var out strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			out.WriteString(paint(pal.diffHead, line))
		case strings.HasPrefix(line, "@@"):
			out.WriteString(paint(pal.diffHunk, line))
		case strings.HasPrefix(line, "-"):
			out.WriteString(paint(pal.diffDel, line))
		case strings.HasPrefix(line, "+"):
			out.WriteString(paint(pal.diffIns, line))
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}
