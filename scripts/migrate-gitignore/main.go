// migrate-gitignore performs the one-time migration to bounded managed blocks.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/project"
	"github.com/YewFence/YewSeal/internal/task"
)

func main() {
	dir := flag.String("dir", ".", "Project directory; a repository is scanned from its root")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments; use -dir PATH")
		os.Exit(2)
	}
	if err := os.Chdir(*dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out io.Writer) (resultErr error) {
	defer func() {
		_, err := fmt.Fprintln(out, "Audit the version-control diff before committing. Legacy sections can contain handwritten rules; restore any that should remain outside the managed block. Run 'yews verify' for the affected project. No commit was created.")
		resultErr = errors.Join(resultErr, err)
	}()
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.DiscoveryDegraded {
		return fmt.Errorf("migration requires complete configuration discovery: %v", cfg.DiscoveryWarnings)
	}
	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{Command: task.ModeClean})
	if err != nil {
		return err
	}
	scopes := config.GroupManagedFilePairs(selection.AllConfigPairs, cfg.LoadedFiles, cfg.CurrentDir)
	owners := make(map[string][]config.FilePair, len(scopes))
	directories := make(map[string]bool)
	for _, scope := range scopes {
		owners[scope.Dir] = config.ResolvedFilePairsToFilePairs(scope.FilePairs)
		directories[scope.Dir] = true
	}
	var failures []error
	err = filepath.WalkDir(cfg.ProjectRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			failures = append(failures, fmt.Errorf("scan %s: %w", path, walkErr))
			return nil
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == ".jj") {
			return filepath.SkipDir
		}
		if entry.Name() == ".gitignore" && !entry.IsDir() {
			directories[filepath.Dir(path)] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	ordered := make([]string, 0, len(directories))
	for dir := range directories {
		ordered = append(ordered, dir)
	}
	sort.Strings(ordered)
	changed, skipped := 0, 0
	for _, dir := range ordered {
		path := filepath.Join(dir, ".gitignore")
		info, err := os.Lstat(path)
		if err == nil && !info.Mode().IsRegular() {
			err = fmt.Errorf("not a regular file")
		}
		if err != nil && !os.IsNotExist(err) {
			_, printErr := fmt.Fprintf(out, "Skipped %s: %v\n", path, err)
			failures = append(failures, err, printErr)
			skipped++
			continue
		}
		pairs, owner := owners[dir]
		modified, err := project.MigrateGitignore(dir, pairs, owner)
		if err != nil {
			_, printErr := fmt.Fprintf(out, "Skipped %s: %v\n", path, err)
			failures = append(failures, fmt.Errorf("%s: %w", path, err), printErr)
			skipped++
			continue
		}
		if modified {
			_, printErr := fmt.Fprintf(out, "Updated %s\n", path)
			failures = append(failures, printErr)
			changed++
		}
	}
	_, printErr := fmt.Fprintf(out, "Migration finished: %d updated, %d skipped.\n", changed, skipped)
	return errors.Join(append(failures, printErr)...)
}
