// Package localdisk implements backup.Destination by writing files under an
// operator-configured base directory (BACKUP_LOCAL_DIR).
package localdisk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Dest writes backup files under baseDir, namespaced per user by
// BackupConfig.LocalSubdir.
type Dest struct {
	baseDir string // absolute, cleaned
}

// New resolves baseDir to an absolute path. It does not require the
// directory to exist yet (created lazily on first write).
func New(baseDir string) (*Dest, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("localdisk: resolve base dir: %w", err)
	}
	return &Dest{baseDir: abs}, nil
}

// Write validates cfg.LocalSubdir cannot escape the base directory, then
// writes data to <base>/<subdir>/<filename>, creating directories as needed.
func (d *Dest) Write(_ context.Context, cfg types.BackupConfig, filename string, data []byte) error {
	dir, err := d.userDir(cfg.LocalSubdir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("localdisk: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, filepath.Base(filename))
	if err := os.WriteFile(path, data, 0o640); err != nil { // #nosec G306 -- backup files, not secrets
		return fmt.Errorf("localdisk: write %s: %w", path, err)
	}
	return nil
}

// List returns the filenames present under cfg.LocalSubdir (base names only,
// no directory component — symmetric with Write's filepath.Base handling).
// Returns an empty slice, not an error, if the directory doesn't exist yet
// (a fresh/never-backed-up user).
func (d *Dest) List(_ context.Context, cfg types.BackupConfig) ([]string, error) {
	dir, err := d.userDir(cfg.LocalSubdir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("localdisk: list %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// Read returns the contents of <base>/<subdir>/<filename>, using the same
// path-escape validation as Write.
func (d *Dest) Read(_ context.Context, cfg types.BackupConfig, filename string) ([]byte, error) {
	dir, err := d.userDir(cfg.LocalSubdir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, filepath.Base(filename))
	data, err := os.ReadFile(path) // #nosec G304 -- filename base-name-only, dir already escape-validated
	if err != nil {
		return nil, fmt.Errorf("localdisk: read %s: %w", path, err)
	}
	return data, nil
}

// userDir resolves subdir against the base directory and rejects anything
// that would escape it (via ".." or an absolute path), before the caller
// ever touches the filesystem. filepath.Join cleans the result, so the only
// remaining check is that the cleaned path still lives under baseDir.
func (d *Dest) userDir(subdir string) (string, error) {
	if subdir == "" {
		return d.baseDir, nil
	}
	joined := filepath.Join(d.baseDir, subdir)
	if joined != d.baseDir && !strings.HasPrefix(joined, d.baseDir+string(filepath.Separator)) {
		return "", fmt.Errorf("localdisk: local_subdir %q escapes base directory", subdir)
	}
	return joined, nil
}
