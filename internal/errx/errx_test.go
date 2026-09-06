package errx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFoundError(t *testing.T) {
	t.Run("defaults to file", func(t *testing.T) {
		err := &NotFoundError{Path: "config.toml"}
		assert.Equal(t, "file config.toml does not exist", err.Error())
	})

	t.Run("uses custom label", func(t *testing.T) {
		err := &NotFoundError{What: "directory", Path: ".age"}
		assert.Equal(t, "directory .age does not exist", err.Error())
	})
}

func TestUnsupportedFormatError(t *testing.T) {
	t.Run("without supported list", func(t *testing.T) {
		err := &UnsupportedFormatError{Path: "config.xml"}
		assert.Equal(t, "unsupported file format for config.xml", err.Error())
	})

	t.Run("with supported list", func(t *testing.T) {
		err := &UnsupportedFormatError{Path: "config.xml", Supported: []string{"yaml", "json"}}
		assert.Equal(t, "unsupported file format for config.xml (supported: yaml, json)", err.Error())
	})
}

func TestKeyFileReadError(t *testing.T) {
	cause := errors.New("permission denied")
	err := &KeyFileReadError{Path: ".age/keys.txt", Err: cause}

	assert.Equal(t, "failed to read key file .age/keys.txt: permission denied", err.Error())
	assert.ErrorIs(t, err, cause)
}

func TestAgeSecretKeyNotFoundError(t *testing.T) {
	err := &AgeSecretKeyNotFoundError{Path: ".age/keys.txt"}
	assert.Equal(t, "no valid Age secret key found in .age/keys.txt", err.Error())
}

func TestExternalCommandError(t *testing.T) {
	cause := errors.New("exit status 1")

	t.Run("includes stderr", func(t *testing.T) {
		err := &ExternalCommandError{
			Op:     "failed to read identity",
			Cmd:    "sh",
			Args:   []string{"-c", "false"},
			Stderr: "boom\n",
			Err:    cause,
		}

		assert.Equal(t, "failed to read identity: exit status 1\nboom", err.Error())
		assert.ErrorIs(t, err, cause)
	})

	t.Run("uses default operation", func(t *testing.T) {
		err := &ExternalCommandError{Err: cause}
		assert.Equal(t, "run command: exit status 1", err.Error())
	})
}

func TestAgeKeyNotFoundError(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		err := &AgeKeyNotFoundError{}
		assert.Equal(t, "no Age key found", err.Error())
	})

	t.Run("with options", func(t *testing.T) {
		err := &AgeKeyNotFoundError{Options: []string{"SOPS_AGE_KEY", ".age/keys.txt"}}
		assert.Equal(t, "no Age key found. Options: SOPS_AGE_KEY, .age/keys.txt", err.Error())
	})
}
