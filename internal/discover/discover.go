package discover

import (
	"io/fs"
	"path/filepath"
	"sort"
)

// Discover returns relative paths under dir that match any include glob and
// no ignore glob. Hidden files/dirs (prefix .) and dirs .git, node_modules,
// vendor are skipped. Result is sorted.
func Discover(dir string, include, ignore []string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		// Skip hidden files/dirs (prefix .)
		if len(base) > 0 && base[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip special dirs
		if d.IsDir() {
			switch base {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		// Files only: match include (any)
		matched := false
		for _, pat := range include {
			ok, _ := filepath.Match(pat, rel)
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
		// Skip if matches any ignore
		for _, pat := range ignore {
			ok, _ := filepath.Match(pat, rel)
			if ok {
				return nil
			}
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
