package seal

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/sopsx"
)

type EncryptOptions struct {
	InputFile      string
	OutputFile     string
	Recipients     []string
	FormatOverride string
	Verbose        bool
	Output         io.Writer
}

type DecryptOptions struct {
	InputFile      string
	OutputFile     string
	IdentityBundle agekey.IdentityBundle
	FormatOverride string
	Verbose        bool
	Force          bool
	Output         io.Writer
}

type DecryptBytesOptions struct {
	InputFile      string
	OutputFile     string
	IdentityBundle agekey.IdentityBundle
	FormatOverride string
	Verbose        bool
	Output         io.Writer
}

type EncryptBytesOptions struct {
	FormatFile     string
	FormatOverride string
	Recipients     []string
	Verbose        bool
	Output         io.Writer
}

func Encrypt(opts EncryptOptions) error {
	out := outputWriter(opts.Output)

	if _, err := os.Stat(opts.InputFile); os.IsNotExist(err) {
		return &errx.NotFoundError{What: "input file", Path: opts.InputFile}
	}

	format, err := resolveFormat(opts.InputFile, opts.FormatOverride)
	if err != nil {
		return err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintf(out, "📖 Reading %s...\n", opts.InputFile)
	}

	plainData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	recipients := append([]string(nil), opts.Recipients...)
	if len(recipients) == 0 {
		return fmt.Errorf("at least one configured age recipient is required")
	}

	encData, err := encryptBytes(plainData, format, EncryptBytesOptions{
		FormatFile:     opts.InputFile,
		FormatOverride: opts.FormatOverride,
		Recipients:     recipients,
		Verbose:        opts.Verbose,
		Output:         opts.Output,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(opts.OutputFile, encData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	_, _ = fmt.Fprintf(out, "✅ Encrypted %s → %s\n", opts.InputFile, opts.OutputFile)
	return nil
}

func EncryptToBytes(plainData []byte, opts EncryptBytesOptions) ([]byte, error) {
	format, err := resolveFormat(opts.FormatFile, opts.FormatOverride)
	if err != nil {
		return nil, err
	}
	return encryptBytes(plainData, format, opts)
}

func encryptBytes(plainData []byte, format string, opts EncryptBytesOptions) ([]byte, error) {
	out := outputWriter(opts.Output)

	if opts.Verbose {
		_, _ = fmt.Fprintln(out, "🔐 Encrypting with SOPS...")
	}

	encData, err := sopsx.Encrypt(plainData, format, opts.Recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return encData, nil
}

func Decrypt(opts DecryptOptions) error {
	out := outputWriter(opts.Output)

	plainData, err := DecryptToBytes(DecryptBytesOptions{
		InputFile:      opts.InputFile,
		OutputFile:     opts.OutputFile,
		IdentityBundle: opts.IdentityBundle,
		FormatOverride: opts.FormatOverride,
		Verbose:        opts.Verbose,
		Output:         opts.Output,
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

	format, err := resolveFormat(opts.OutputFile, opts.FormatOverride)
	if err != nil {
		return nil, err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintf(out, "📖 Reading %s...\n", opts.InputFile)
		_, _ = fmt.Fprintln(out, "🔓 Decrypting with SOPS...")
	}

	if len(opts.IdentityBundle.Identities()) == 0 {
		return nil, fmt.Errorf("identity bundle is required")
	}
	privateKey := opts.IdentityBundle.String()

	encData, err := os.ReadFile(opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	plainData, err := sopsx.Decrypt(encData, format, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plainData, nil
}

func ExtractAgeRecipientsFromEncryptedFile(encryptedFile, formatFile, formatOverride string) ([]string, error) {
	format, err := resolveFormat(formatFile, formatOverride)
	if err != nil {
		return nil, err
	}

	encData, err := os.ReadFile(encryptedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	recipients, err := sopsx.ExtractAgeRecipients(encData, format)
	if err != nil {
		return nil, fmt.Errorf("failed to extract public keys from encrypted file: %w", err)
	}
	return recipients, nil
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

func outputWriter(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return os.Stdout
}
