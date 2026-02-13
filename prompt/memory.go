package prompt

import (
	"os"
	"path/filepath"
	"strings"
)

// PromptEntry is a saved prompt in the prompts directory (flat or in a folder).
type PromptEntry struct {
	Name   string // template name (no .yaml)
	Folder string // subdir name, or empty if at root
}

// ListPromptsInDir lists prompts under dir: flat .yaml files and one level of subdirs with .yaml files.
func ListPromptsInDir(dir string) ([]PromptEntry, error) {
	var out []PromptEntry
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			folder := e.Name()
			if !IsValidTemplateName(folder) {
				continue
			}
			for _, se := range subEntries {
				if se.IsDir() || !strings.HasSuffix(se.Name(), ".yaml") {
					continue
				}
				name := strings.TrimSuffix(se.Name(), ".yaml")
				if IsValidTemplateName(name) {
					out = append(out, PromptEntry{Name: name, Folder: folder})
				}
			}
			continue
		}
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if IsValidTemplateName(name) {
			out = append(out, PromptEntry{Name: name, Folder: ""})
		}
	}
	return out, nil
}
