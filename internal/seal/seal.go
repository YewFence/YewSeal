package seal

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/YewFence/YewSeal/internal/agekey"
	tools "github.com/YewFence/YewSeal/internal/doctor"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/fileformat"
	"github.com/YewFence/YewSeal/internal/remarshal"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

type EncryptOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	PublicKey      string
	FormatOverride string
	Verbose        bool
}

type DecryptOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	FormatOverride string
	Verbose        bool
	Force          bool
}

type DecryptBytesOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	FormatOverride string
	Verbose        bool
}

type EditOptions struct {
	File    string
	Editor  string
	KeyFile string
}

func Encrypt(opts EncryptOptions) error {
	if _, err := os.Stat(opts.InputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: opts.InputFile}
	}

	inputFormat := fileformat.Resolve(opts.InputFile, opts.FormatOverride)
	if inputFormat == fileformat.Unknown {
		return &errx.UnsupportedFormatError{Path: opts.InputFile, Supported: fileformat.SupportedExtensions()}
	}
	if fileformat.NeedsConversion(inputFormat) {
		if err := tools.CheckRemarshal(); err != nil {
			return err
		}
	}

	if opts.Verbose {
		fmt.Printf("📖 Reading %s...\n", opts.InputFile)
	}

	plainData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if fileformat.NeedsConversion(inputFormat) {
		if opts.Verbose {
			fmt.Println("🔄 Converting TOML to YAML...")
		}
		plainData, err = remarshal.TOMLToYAML(plainData)
		if err != nil {
			return fmt.Errorf("failed to convert TOML to YAML: %w", err)
		}
	}

	if opts.Verbose {
		fmt.Println("🔐 Encrypting with SOPS...")
	}

	publicKey, err := agekey.GetPublicKey(opts.PublicKey, opts.KeyFile, opts.Verbose)
	if err != nil {
		return err
	}

	encData, err := sopsx.Encrypt(plainData, fileformat.SOPSFormat(inputFormat), publicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := os.WriteFile(opts.OutputFile, encData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Encrypted %s → %s\n", opts.InputFile, opts.OutputFile)
	return nil
}

func Decrypt(opts DecryptOptions) error {
	plainData, err := DecryptToBytes(DecryptBytesOptions{
		InputFile:      opts.InputFile,
		OutputFile:     opts.OutputFile,
		KeyFile:        opts.KeyFile,
		FormatOverride: opts.FormatOverride,
		Verbose:        opts.Verbose,
	})
	if err != nil {
		return err
	}

	if err := writeDecryptedFile(opts.InputFile, opts.OutputFile, plainData, opts.Force); err != nil {
		return err
	}

	fmt.Printf("✅ Decrypted %s → %s\n", opts.InputFile, opts.OutputFile)
	return nil
}

func DecryptToBytes(opts DecryptBytesOptions) ([]byte, error) {
	if _, err := os.Stat(opts.InputFile); os.IsNotExist(err) {
		return nil, &errx.NotFoundError{What: "input file", Path: opts.InputFile}
	}

	outputFormat := fileformat.Resolve(opts.OutputFile, opts.FormatOverride)
	if outputFormat == fileformat.Unknown {
		return nil, &errx.UnsupportedFormatError{Path: opts.OutputFile, Supported: fileformat.SupportedExtensions()}
	}
	if fileformat.NeedsConversion(outputFormat) {
		if err := tools.CheckRemarshal(); err != nil {
			return nil, err
		}
	}

	if opts.Verbose {
		fmt.Printf("📖 Reading %s...\n", opts.InputFile)
		fmt.Println("🔓 Decrypting with SOPS...")
	}

	privateKey, err := agekey.GetAgeKey(opts.KeyFile)
	if err != nil {
		return nil, err
	}

	encData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	plainData, err := sopsx.Decrypt(encData, fileformat.SOPSFormat(outputFormat), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	if fileformat.NeedsConversion(outputFormat) {
		if opts.Verbose {
			fmt.Println("🔄 Converting YAML to TOML...")
		}
		plainData, err = remarshal.YAMLToTOML(plainData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to TOML: %w", err)
		}
	}

	return plainData, nil
}

func Edit(opts EditOptions) error {
	if _, err := os.Stat(opts.File); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: opts.File}
	}

	editFormat := fileformat.Detect(opts.File)
	if editFormat == fileformat.Unknown {
		editFormat = fileformat.YAML
	}
	sopsType := fileformat.SOPSFormat(editFormat)

	key, err := agekey.GetAgeKey(opts.KeyFile)
	if err != nil {
		return err
	}

	encData, err := os.ReadFile(opts.File)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	plainData, err := sopsx.Decrypt(encData, sopsType, key)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "yews-edit-*."+string(editFormat))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(plainData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	originalHash := sha256.Sum256(plainData)
	editorCmd := resolveEditor(opts.Editor)

	fmt.Printf("✏️  Opening %s in %s...\n", opts.File, editorCmd)

	parts := strings.Fields(editorCmd)
	args := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	editedData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	editedHash := sha256.Sum256(editedData)
	if originalHash == editedHash {
		fmt.Println("⏭️  No changes detected, skipping re-encryption")
		return nil
	}

	store := sopsx.StoreForFormat(sopsType)
	tree, err := store.LoadEncryptedFile(encData)
	if err != nil {
		return fmt.Errorf("failed to parse encrypted file metadata: %w", err)
	}

	publicKey, err := sopsx.ExtractAgeRecipientFromTree(tree)
	if err != nil {
		return fmt.Errorf("failed to extract public key from encrypted file: %w", err)
	}

	newEncData, err := sopsx.Encrypt(editedData, sopsType, publicKey)
	if err != nil {
		return fmt.Errorf("failed to re-encrypt: %w", err)
	}

	if err := os.WriteFile(opts.File, newEncData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	fmt.Println("✅ File edited and re-encrypted successfully")
	return nil
}

func writeDecryptedFile(inputFile, outputFile string, plainData []byte, force bool) error {
	currentData, err := os.ReadFile(outputFile)
	if err == nil {
		if bytes.Equal(currentData, plainData) {
			return nil
		}
		if !force {
			return &errx.ProtectedOverwriteError{SourceFile: inputFile, TargetFile: outputFile}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	if err := os.WriteFile(outputFile, plainData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

func resolveEditor(editor string) string {
	if editor != "" {
		return editor
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if runtime.GOOS == "windows" {
		for _, candidate := range []string{"code -w", "notepad"} {
			name := strings.Fields(candidate)[0]
			if p, _ := exec.LookPath(name); p != "" {
				return candidate
			}
		}
		return "notepad"
	}
	return "vi"
}
