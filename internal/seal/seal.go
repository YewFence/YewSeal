package seal

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/sopsx"
	"github.com/google/shlex"
)

type EncryptOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	PublicKey      string
	FormatOverride string
	Verbose        bool
	Output         io.Writer
	Warnings       io.Writer
}

type DecryptOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	FormatOverride string
	Verbose        bool
	Force          bool
	Output         io.Writer
	Warnings       io.Writer
}

type DecryptBytesOptions struct {
	InputFile      string
	OutputFile     string
	KeyFile        string
	FormatOverride string
	Verbose        bool
	Output         io.Writer
	Warnings       io.Writer
}

type EditOptions struct {
	File     string
	Editor   string
	KeyFile  string
	Output   io.Writer
	Warnings io.Writer
}

func Encrypt(opts EncryptOptions) error {
	out := outputWriter(opts.Output)

	if _, err := os.Stat(opts.InputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: opts.InputFile}
	}

	plan, err := newCodecPlan(opts.InputFile, opts.FormatOverride)
	if err != nil {
		return err
	}
	if err := plan.checkTools(); err != nil {
		return err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintf(out, "📖 Reading %s...\n", opts.InputFile)
	}

	plainData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	if plan.needsRemarshal {
		if opts.Verbose {
			_, _ = fmt.Fprintln(out, plan.encryptAction)
		}
	}
	plainData, err = plan.prepareEncrypt(plainData)
	if err != nil {
		return err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintln(out, "🔐 Encrypting with SOPS...")
	}

	publicKey, err := agekey.GetPublicKeyWithOutput(opts.PublicKey, opts.KeyFile, opts.Verbose, out)
	if err != nil {
		return err
	}

	encData, err := sopsx.Encrypt(plainData, plan.sopsFormat, publicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := os.WriteFile(opts.OutputFile, encData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	_, _ = fmt.Fprintf(out, "✅ Encrypted %s → %s\n", opts.InputFile, opts.OutputFile)
	return nil
}

func Decrypt(opts DecryptOptions) error {
	out := outputWriter(opts.Output)

	plainData, err := DecryptToBytes(DecryptBytesOptions{
		InputFile:      opts.InputFile,
		OutputFile:     opts.OutputFile,
		KeyFile:        opts.KeyFile,
		FormatOverride: opts.FormatOverride,
		Verbose:        opts.Verbose,
		Output:         opts.Output,
		Warnings:       opts.Warnings,
	})
	if err != nil {
		return err
	}

	if err := writeDecryptedFile(opts.InputFile, opts.OutputFile, plainData, opts.Force); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "✅ Decrypted %s → %s\n", opts.InputFile, opts.OutputFile)
	return nil
}

func DecryptToBytes(opts DecryptBytesOptions) ([]byte, error) {
	out := outputWriter(opts.Output)

	if _, err := os.Stat(opts.InputFile); os.IsNotExist(err) {
		return nil, &errx.NotFoundError{What: "input file", Path: opts.InputFile}
	}

	plan, err := newCodecPlan(opts.OutputFile, opts.FormatOverride)
	if err != nil {
		return nil, err
	}
	if err := plan.checkTools(); err != nil {
		return nil, err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintf(out, "📖 Reading %s...\n", opts.InputFile)
		_, _ = fmt.Fprintln(out, "🔓 Decrypting with SOPS...")
	}

	privateKey, err := agekey.GetAgeKey(opts.KeyFile)
	if err != nil {
		return nil, err
	}

	encData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	plainData, err := sopsx.Decrypt(encData, plan.sopsFormat, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	if plan.needsRemarshal {
		if opts.Verbose {
			_, _ = fmt.Fprintln(out, plan.decryptAction)
		}
	}
	plainData, err = plan.restoreDecrypt(plainData)
	if err != nil {
		return nil, err
	}

	return plainData, nil
}

func Edit(opts EditOptions) error {
	out := outputWriter(opts.Output)

	if _, err := os.Stat(opts.File); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "file", Path: opts.File}
	}

	editFormat := detectFormat(opts.File)
	if editFormat == formatUnknown {
		editFormat = formatYAML
	}
	sopsType := codecPlanForFormat(editFormat).sopsFormat

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
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(plainData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	originalHash := sha256.Sum256(plainData)
	editorCmd := resolveEditor(opts.Editor)

	_, _ = fmt.Fprintf(out, "✏️  Opening %s in %s...\n", opts.File, editorCmd)

	parts, err := splitEditorCommand(editorCmd)
	if err != nil {
		return err
	}
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
		_, _ = fmt.Fprintln(out, "⏭️  No changes detected, skipping re-encryption")
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

	_, _ = fmt.Fprintln(out, "✅ File edited and re-encrypted successfully")
	return nil
}

func writeDecryptedFile(inputFile, outputFile string, plainData []byte, force bool) error {
	currentData, err := os.ReadFile(outputFile)
	if err == nil {
		if bytes.Equal(currentData, plainData) {
			if err := os.Chmod(outputFile, 0600); err != nil {
				return fmt.Errorf("failed to set output file permissions: %w", err)
			}
			return nil
		}
		if !force {
			return &errx.ProtectedOverwriteError{SourceFile: inputFile, TargetFile: outputFile}
		}
		if err := os.Chmod(outputFile, 0600); err != nil {
			return fmt.Errorf("failed to set output file permissions: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	if err := os.WriteFile(outputFile, plainData, 0600); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	if err := os.Chmod(outputFile, 0600); err != nil {
		return fmt.Errorf("failed to set output file permissions: %w", err)
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
			parts, err := splitEditorCommand(candidate)
			if err != nil || len(parts) == 0 {
				continue
			}
			name := parts[0]
			if p, _ := exec.LookPath(name); p != "" {
				return candidate
			}
		}
		return "notepad"
	}
	return "vi"
}

func splitEditorCommand(editorCmd string) ([]string, error) {
	parts, err := shlex.Split(editorCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse editor command: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor command is empty")
	}
	return parts, nil
}

func outputWriter(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return os.Stdout
}
