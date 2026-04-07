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

func TestUnknownProviderError(t *testing.T) {
	t.Run("without supported list", func(t *testing.T) {
		err := &UnknownProviderError{Name: "vault"}
		assert.Equal(t, "unknown provider: vault", err.Error())
	})

	t.Run("with supported list", func(t *testing.T) {
		err := &UnknownProviderError{Name: "vault", Supported: []string{"infisical"}}
		assert.Equal(t, "unknown provider: vault (supported: infisical)", err.Error())
	})
}

func TestKeyFileNotFoundError(t *testing.T) {
	err := &KeyFileNotFoundError{Path: ".age/keys.txt"}
	assert.Equal(t, "key file not found: .age/keys.txt\nRun 'yews init' first to generate keys", err.Error())
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

func TestMissingDependencyError(t *testing.T) {
	t.Run("without install hint", func(t *testing.T) {
		err := &MissingDependencyError{Name: "infisical"}
		assert.Equal(t, "infisical not found", err.Error())
	})

	t.Run("with install hint", func(t *testing.T) {
		err := &MissingDependencyError{Name: "infisical", InstallHint: "https://example.com/install"}
		assert.Equal(t, "infisical not found\nInstall: https://example.com/install", err.Error())
	})
}

func TestMissingProjectConfigError(t *testing.T) {
	t.Run("without hint", func(t *testing.T) {
		err := &MissingProjectConfigError{Path: ".infisical.json"}
		assert.Equal(t, ".infisical.json not found", err.Error())
	})

	t.Run("with hint", func(t *testing.T) {
		err := &MissingProjectConfigError{Path: ".infisical.json", Hint: "Run 'infisical init' first"}
		assert.Equal(t, ".infisical.json not found\nRun 'infisical init' first", err.Error())
	})
}

func TestExternalCommandError(t *testing.T) {
	cause := errors.New("exit status 1")

	t.Run("includes stderr", func(t *testing.T) {
		err := &ExternalCommandError{
			Op:     "failed to sync",
			Cmd:    "infisical",
			Args:   []string{"secrets", "set"},
			Stderr: "boom\n",
			Err:    cause,
		}

		assert.Equal(t, "failed to sync: exit status 1\nboom", err.Error())
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

func TestPublicKeyNotFoundError(t *testing.T) {
	t.Run("uses default tried list with cause", func(t *testing.T) {
		cause := errors.New("invalid key")
		err := &PublicKeyNotFoundError{
			KeyFile: ".age/keys.txt",
			Cause:   cause,
		}

		assert.Equal(
			t,
			"no public key found. Tried: CLI parameter, SOPS_AGE_RECIPIENTS env, .yewseal.toml config, and extracting from private key file, and extracting from .age/keys.txt (failed: invalid key). Please run 'yews init' or provide --public-key",
			err.Error(),
		)
		assert.ErrorIs(t, err, cause)
	})

	t.Run("uses custom tried list without cause", func(t *testing.T) {
		err := &PublicKeyNotFoundError{
			KeyFile: ".age/keys.txt",
			Tried:   []string{"CLI parameter", "config file"},
		}

		assert.Equal(
			t,
			"no public key found. Tried: CLI parameter, config file, and extracting from .age/keys.txt (no valid public key found). Please run 'yews init' or provide --public-key",
			err.Error(),
		)
	})
}
