package main

import (
	"testing"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCLIFormatOverride_Empty(t *testing.T) {
	format, err := validateCLIFormatOverride("")
	require.NoError(t, err)
	assert.Equal(t, "", format)
}

func TestValidateCLIFormatOverride_NormalizesAlias(t *testing.T) {
	format, err := validateCLIFormatOverride("dotenv")
	require.NoError(t, err)
	assert.Equal(t, "env", format)
}

func TestValidateCLIFormatOverride_Invalid(t *testing.T) {
	_, err := validateCLIFormatOverride("xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestResolveFormatOverride_UsesConfigValue(t *testing.T) {
	format, err := resolveFormatOverride("", config.FilePair{Format: "env"})
	require.NoError(t, err)
	assert.Equal(t, "env", format)
}

func TestResolveFormatOverride_PrefersCLIValue(t *testing.T) {
	format, err := resolveFormatOverride("json", config.FilePair{Format: "env"})
	require.NoError(t, err)
	assert.Equal(t, "json", format)
}
