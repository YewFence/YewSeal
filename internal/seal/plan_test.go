package seal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodecPlan(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		override       string
		wantUserFormat format
		wantSOPSFormat string
		wantRemarshal  bool
	}{
		{
			name:           "toml uses yaml payload",
			path:           "config.toml",
			wantUserFormat: formatTOML,
			wantSOPSFormat: "yaml",
			wantRemarshal:  true,
		},
		{
			name:           "yaml uses yaml payload",
			path:           "config.yaml",
			wantUserFormat: formatYAML,
			wantSOPSFormat: "yaml",
		},
		{
			name:           "json uses json payload",
			path:           "config.json",
			wantUserFormat: formatJSON,
			wantSOPSFormat: "json",
		},
		{
			name:           "env uses dotenv payload",
			path:           "config.env",
			wantUserFormat: formatENV,
			wantSOPSFormat: "dotenv",
		},
		{
			name:           "ini uses ini payload",
			path:           "config.ini",
			wantUserFormat: formatINI,
			wantSOPSFormat: "ini",
		},
		{
			name:           "binary uses binary payload",
			path:           "config.bin",
			wantUserFormat: formatBinary,
			wantSOPSFormat: "binary",
		},
		{
			name:           "override wins over path",
			path:           "config.unknown",
			override:       "json",
			wantUserFormat: formatJSON,
			wantSOPSFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := newCodecPlan(tt.path, tt.override)
			require.NoError(t, err)
			assert.Equal(t, tt.wantUserFormat, plan.userFormat)
			assert.Equal(t, tt.wantSOPSFormat, plan.sopsFormat)
			assert.Equal(t, tt.wantRemarshal, plan.needsRemarshal)
		})
	}
}

func TestNewCodecPlanFallsBackToBinaryForUnknownDetectedFormat(t *testing.T) {
	plan, err := newCodecPlan("config.unknown", "")

	require.NoError(t, err)
	assert.Equal(t, formatBinary, plan.userFormat)
	assert.Equal(t, "binary", plan.sopsFormat)
	assert.Contains(t, plan.warning, "using binary format")
}

func TestNewCodecPlanRejectsInvalidExplicitOverride(t *testing.T) {
	_, err := newCodecPlan("config.yaml", "xml")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported format override "xml"`)
}

func TestCodecPlanForFormatUnknownFallsBackToBinary(t *testing.T) {
	plan := codecPlanForFormat(formatUnknown)

	assert.Equal(t, formatUnknown, plan.userFormat)
	assert.Equal(t, "binary", plan.sopsFormat)
	assert.False(t, plan.needsRemarshal)
}
