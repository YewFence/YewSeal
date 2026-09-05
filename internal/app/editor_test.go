package app

import (
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitEditorCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "vim", []string{"vim"}},
		{"whitespace", " \tcode  --wait\r\n", []string{"code", "--wait"}},
		{"single quotes", `editor 'a b'`, []string{"editor", "a b"}},
		{"empty arguments", `editor "" ''`, []string{"editor", "", ""}},
		{"fragments", `ed"it"'or' a" b"c`, []string{"editor", "a bc"}},
		{"escaped space", `some\ editor a\ b`, []string{"some editor", "a b"}},
		{"escaped quotes", `editor \" \' \\`, []string{"editor", `"`, `'`, `\`}},
		{"double quote escapes", `editor "a\"b\\c\d"`, []string{"editor", `a"b\c\d`}},
		{"single quote literals", `editor 'a\b"c'`, []string{"editor", `a\b"c`}},
		{"windows path", `"C:\Program Files\Editor\editor.exe" --wait`, []string{`C:\Program Files\Editor\editor.exe`, "--wait"}},
		{"unicode", "editor '\u4e91\u67ab \u914d\u7f6e'", []string{"editor", "\u4e91\u67ab \u914d\u7f6e"}},
		{"quoted shell literals", "editor '$HOME' \"$(echo x)\" '`id`' 'a|b'", []string{"editor", "$HOME", "$(echo x)", "`id`", "a|b"}},
		{"escaped operator", `editor a\;b`, []string{"editor", "a;b"}},
		{"no glob or comment expansion", `editor * ~ #tag`, []string{"editor", "*", "~", "#tag"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitEditorCommand(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitEditorCommandErrors(t *testing.T) {
	for _, input := range []string{"", " \t\n", `"" --wait`, `''`, `editor 'open`, `editor "open`, `editor \`, "editor\x00arg", "editor\xff"} {
		t.Run(input, func(t *testing.T) {
			parts, err := splitEditorCommand(input)
			require.Error(t, err)
			assert.Nil(t, parts)
		})
	}
	for _, syntax := range []string{"|", "&&", ";", ">", "<", "(", ")", "$HOME", "$(id)", "`id`"} {
		t.Run(syntax, func(t *testing.T) {
			parts, err := splitEditorCommand("editor " + syntax)
			require.ErrorContains(t, err, "unsupported shell syntax")
			assert.Nil(t, parts)
		})
	}
}

func TestResolveEditor(t *testing.T) {
	t.Setenv("VISUAL", "visual --wait")
	t.Setenv("EDITOR", "editor")
	assert.Equal(t, "visual --wait", resolveEditor())
	t.Setenv("VISUAL", "")
	assert.Equal(t, "editor", resolveEditor())
	t.Setenv("EDITOR", "")
	want := "vi"
	if runtime.GOOS == "windows" {
		want = "notepad"
	}
	assert.Equal(t, want, resolveEditor())
}

func FuzzSplitEditorCommand(f *testing.F) {
	for _, seed := range []string{"", "a b", `'\"`, "\u4e91\u67ab", "$(id)", "\x00", "\xff"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		parts, err := splitEditorCommand(arg)
		if err == nil {
			require.NotEmpty(t, parts)
			require.NotEmpty(t, parts[0])
		}
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return
		}
		quoted := "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
		parts, err = splitEditorCommand("editor " + quoted)
		require.NoError(t, err)
		assert.Equal(t, []string{"editor", arg}, parts)
	})
}
