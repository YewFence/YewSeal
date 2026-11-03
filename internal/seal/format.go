package seal

import (
	"fmt"
	"strings"

	"github.com/YewFence/YewSeal/internal/fileformat"
)

func resolveFormat(path, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		format, ok := fileformat.Normalize(override)
		if !ok {
			return "", fmt.Errorf("unsupported format override %q (supported: %s)", override, fileformat.Supported())
		}
		return format, nil
	}
	format, ok := fileformat.Detect(path)
	if !ok {
		return "", fmt.Errorf("could not detect format for %s (supported: %s). Declare a format in .yewseal.toml; use binary for arbitrary content", path, fileformat.Supported())
	}
	return format, nil
}
