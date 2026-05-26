package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/prompt-ctl/promptctl/internal/discover"
	"github.com/prompt-ctl/promptctl/internal/scoreconfig"
	"github.com/prompt-ctl/promptctl/prompt"
)

func runFix() error {
	args := os.Args[2:]

	var suggest, dryRun bool
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--suggest":
			suggest = true
		case "--dry-run":
			dryRun = true
		default:
			if !strings.HasPrefix(arg, "-") {
				positionals = append(positionals, arg)
			}
		}
	}

	// Load score config (same as runScore: walk up to find .promptctl, scoreconfig.Load)
	var cfg scoreconfig.ScoreConfig
	promptctlPath := findScoreConfigDir()
	if promptctlPath != "" {
		cfg = scoreconfig.Load(filepath.Dir(promptctlPath))
	} else {
		cfg = scoreconfig.DefaultConfig()
	}

	include := cfg.Include
	if len(include) == 0 {
		include = []string{"*.txt", "*.md"}
	}
	ignore := cfg.Ignore

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to get working directory:", err)
		os.Exit(2)
	}

	// Resolve file list: positionals, or config Dirs, or "."
	var files []string
	seen := make(map[string]bool)

	addFile := func(path string) {
		abs, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid path %q: %v\n", path, err)
			return
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		files = append(files, abs)
	}

	if len(positionals) > 0 {
		for _, p := range positionals {
			abs, err := filepath.Abs(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid path %q: %v\n", p, err)
				os.Exit(2)
			}
			info, err := os.Stat(abs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(2)
			}
			if info.IsDir() {
				discovered, err := discover.Discover(abs, include, ignore)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: discover %q: %v\n", p, err)
					os.Exit(2)
				}
				for _, rel := range discovered {
					addFile(filepath.Join(abs, rel))
				}
			} else {
				addFile(abs)
			}
		}
	} else if len(cfg.Dirs) > 0 {
		for _, d := range cfg.Dirs {
			abs, err := filepath.Abs(d)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid dir %q: %v\n", d, err)
				os.Exit(2)
			}
			discovered, err := discover.Discover(abs, include, ignore)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: discover %q: %v\n", d, err)
				os.Exit(2)
			}
			for _, rel := range discovered {
				addFile(filepath.Join(abs, rel))
			}
		}
	} else {
		discovered, err := discover.Discover(".", include, ignore)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: discover .: %v\n", err)
			os.Exit(2)
		}
		for _, rel := range discovered {
			addFile(filepath.Join(cwd, rel))
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No prompt files found.")
		return nil
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skip %s: %v\n", f, err)
			continue
		}
		if !utf8.Valid(content) {
			fmt.Fprintf(os.Stderr, "Warning: skip %s: not valid UTF-8\n", f)
			continue
		}
		s := string(content)
		newContent := prompt.ApplyStructure(prompt.ApplyFormat(s))

		rel, _ := filepath.Rel(cwd, f)
		if rel == "" || strings.HasPrefix(rel, "..") {
			rel = f
		}

		if suggest {
			suggestion, err := prompt.Suggest(s, "scope")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: suggest for %s: %v\n", rel, err)
			} else if suggestion != "" {
				fmt.Printf("%s: scope: %s\n", rel, suggestion)
			}
		}

		if dryRun {
			fmt.Printf("=== %s ===\n%s\n", rel, newContent)
			continue
		}

		if err := os.WriteFile(f, []byte(newContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: write %s: %v\n", rel, err)
			os.Exit(2)
		}
	}

	return nil
}
