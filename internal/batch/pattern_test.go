package batch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternMatcherDecision(t *testing.T) {
	matcher, err := NewPatternMatcher([]string{
		"# comment",
		"*.toml",
		"**/*.env",
		"!*.example.toml",
		"/root.yaml",
		"secrets/",
		`\#literal`,
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		path         string
		isDir        bool
		wantDecided  bool
		wantIncluded bool
	}{
		{name: "basename any depth", path: "config/app.toml", wantDecided: true, wantIncluded: true},
		{name: "last match excludes", path: "app.example.toml", wantDecided: true, wantIncluded: false},
		{name: "globstar", path: "config/dev/app.env", wantDecided: true, wantIncluded: true},
		{name: "anchored match", path: "root.yaml", wantDecided: true, wantIncluded: true},
		{name: "anchored miss", path: "config/root.yaml", wantDecided: false, wantIncluded: false},
		{name: "directory only", path: "secrets", isDir: true, wantDecided: true, wantIncluded: true},
		{name: "directory only misses file", path: "secrets", wantDecided: false, wantIncluded: false},
		{name: "literal hash", path: "#literal", wantDecided: true, wantIncluded: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decided, included := matcher.Decision(tt.path, tt.isDir)
			assert.Equal(t, tt.wantDecided, decided)
			assert.Equal(t, tt.wantIncluded, included)
		})
	}
}

func TestParsePatternRulesRejectsInvalidNegation(t *testing.T) {
	_, err := ParsePatternRules([]string{"!"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing pattern after negation")
}
