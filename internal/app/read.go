package app

import (
	"fmt"
	"io"

	"github.com/YewFence/YewSeal/internal/agekey"
	"github.com/YewFence/YewSeal/internal/config"
)

func prepareRead(w io.Writer, cfg *config.Config, opts config.SelectionOptions, keyFile string, verbose bool) (config.ResolvedSelection, agekey.IdentityBundle, error) {
	selection, err := config.ResolveSelection(cfg, opts)
	if err != nil {
		return selection, agekey.IdentityBundle{}, err
	}
	bundle, err := agekey.GetIdentityBundle(keyFile)
	if err != nil {
		return selection, bundle, err
	}
	for _, pair := range selection.FilePairs {
		if pair.RecipientWarning != "" {
			if _, err := fmt.Fprintln(w, pair.RecipientWarning); err != nil {
				return selection, bundle, err
			}
		}
	}
	if verbose {
		if _, err := fmt.Fprintf(w, "Selected %d file pairs from %d config files\n", len(selection.FilePairs), len(selection.ConfigFiles)); err != nil {
			return selection, bundle, err
		}
		for _, pair := range selection.FilePairs {
			if _, err := fmt.Fprintf(w, "  %s -> %s\n", config.DisplayPath(config.CurrentDir(cfg), pair.PlaintextPath), config.DisplayPath(config.CurrentDir(cfg), pair.EncryptedPath)); err != nil {
				return selection, bundle, err
			}
		}
	}
	return selection, bundle, nil
}

// Output failures stop the command rather than becoming per-file failures.
type diagnosticWriter struct {
	io.Writer
	err error
}

func (w *diagnosticWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}
