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
	_, err := o.Write([]byte(highlightUnifiedDiff(text, colorEnabled)))
	return err
}

func highlightUnifiedDiff(text string, enabled bool) string {
	if !enabled || text == "" { return text }
	headerColor := color.New(color.FgCyan, color.Bold)
	hunkColor := color.New(color.FgMagenta)
	deleteColor := color.New(color.FgRed)
	insertColor := color.New(color.FgGreen)
	for _, c := range []*color.Color{headerColor, hunkColor, deleteColor, insertColor} { c.EnableColor() }
	var out strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			out.WriteString(headerColor.Sprint(line))
		case strings.HasPrefix(line, "@@"):
			out.WriteString(hunkColor.Sprint(line))
		case strings.HasPrefix(line, "-"):
			out.WriteString(deleteColor.Sprint(line))
		case strings.HasPrefix(line, "+"):
			out.WriteString(insertColor.Sprint(line))
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}
