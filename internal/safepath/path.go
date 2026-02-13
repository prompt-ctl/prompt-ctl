package safepath

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathOutsideBase is returned when the resolved path is not under the allowed base.
var ErrPathOutsideBase = errors.New("path is outside allowed base directory")

// SafeFilePath resolves path relative to base (typically cwd) and returns a clean absolute path
// if it lies under base. Prevents path traversal (e.g. ../../../etc/passwd).
func SafeFilePath(base, path string) (string, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	baseAbs = filepath.Clean(baseAbs)

	resolved := path
	if !filepath.IsAbs(path) {
		resolved = filepath.Join(baseAbs, path)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	if !underBase(abs, baseAbs) {
		return "", ErrPathOutsideBase
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("path is a directory, not a file")
	}
	return abs, nil
}

// SafeDirPath resolves path relative to base and returns a clean absolute path
// if it lies under base and is a directory.
func SafeDirPath(base, path string) (string, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	baseAbs = filepath.Clean(baseAbs)

	resolved := path
	if !filepath.IsAbs(path) {
		resolved = filepath.Join(baseAbs, path)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	if !underBase(abs, baseAbs) {
		return "", ErrPathOutsideBase
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return abs, nil
}

// underBase returns true if path is under base (or equal to base).
func underBase(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
