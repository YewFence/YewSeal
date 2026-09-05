package app

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"
)

func resolveEditor() string {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if editor := os.Getenv(name); editor != "" {
			return editor
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// splitEditorCommand parses arguments, not shell programs. Quoted fragments
// concatenate; single quotes are literal, and double quotes only escape " and \.
func splitEditorCommand(command string) ([]string, error) {
	if !utf8.ValidString(command) || strings.ContainsRune(command, 0) {
		return nil, fmt.Errorf("failed to parse editor command: invalid UTF-8 or NUL byte")
	}
	var parts []string
	var word strings.Builder
	var quote byte
	started := false
	flush := func() {
		if started {
			parts = append(parts, word.String())
			word.Reset()
			started = false
		}
	}
	for i := 0; i < len(command); i++ {
		c := command[i]
		if quote != 0 {
			switch {
			case c == quote:
				quote = 0
			case quote == '"' && c == '\\' && i+1 < len(command) && (command[i+1] == '"' || command[i+1] == '\\'):
				i++
				word.WriteByte(command[i])
			default:
				word.WriteByte(c)
			}
			continue
		}
		switch c {
		case ' ', '\t', '\n', '\r':
			flush()
		case '\'', '"':
			quote = c
			started = true
		case '\\':
			i++
			if i == len(command) {
				return nil, fmt.Errorf("failed to parse editor command: trailing escape")
			}
			word.WriteByte(command[i])
			started = true
		case '|', '&', ';', '<', '>', '(', ')', '$', '`':
			return nil, fmt.Errorf("failed to parse editor command: unsupported shell syntax %q; quote literal arguments", c)
		default:
			word.WriteByte(c)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("failed to parse editor command: unterminated quote")
	}
	flush()
	if len(parts) == 0 || parts[0] == "" {
		return nil, fmt.Errorf("editor command is empty")
	}
	return parts, nil
}
