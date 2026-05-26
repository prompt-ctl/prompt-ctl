package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/prompt-ctl/promptctl/internal/discover"
	"github.com/prompt-ctl/promptctl/internal/scoreconfig"
	"github.com/prompt-ctl/promptctl/prompt"
)

// findScoreConfigDir walks up from cwd to find a directory containing .promptctl (same logic as config.findLocalConfig).
// Returns the absolute path of the .promptctl directory, or "" if not found.
func findScoreConfigDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".promptctl")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func runScore() error {
	args := os.Args[2:]

	// Parse flags: --min-score / -min-score, --format
	var minScoreFlag *int
	formatJSON := false
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--format=json":
			formatJSON = true
		case arg == "--format" && i+1 < len(args):
			if args[i+1] == "json" {
				formatJSON = true
			}
			i++
		case strings.HasPrefix(arg, "--min-score=") || strings.HasPrefix(arg, "-min-score="):
			var n int
			if _, err := fmt.Sscanf(strings.TrimPrefix(strings.TrimPrefix(arg, "--min-score="), "-min-score="), "%d", &n); err == nil {
				minScoreFlag = &n
			}
		case (arg == "--min-score" || arg == "-min-score") && i+1 < len(args):
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err == nil {
				minScoreFlag = &n
			}
			i++
		default:
			if !strings.HasPrefix(arg, "-") {
				positionals = append(positionals, arg)
			}
		}
	}

	// Resolve score config
	var cfg scoreconfig.ScoreConfig
	promptctlPath := findScoreConfigDir()
	if promptctlPath != "" {
		cfg = scoreconfig.Load(filepath.Dir(promptctlPath))
	} else {
		cfg = scoreconfig.DefaultConfig()
	}

	minScore := cfg.MinScore
	if minScore == 0 {
		minScore = 80
	}
	if minScoreFlag != nil {
		minScore = *minScoreFlag
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

	// Resolve file list: flatten to absolute paths
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
		if formatJSON {
			out := struct {
				Files    []interface{} `json:"files"`
				MinScore int           `json:"min_score"`
				OK       bool          `json:"ok"`
			}{nil, minScore, false}
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(out)
		}
		os.Exit(1)
	}

	type fileResult struct {
		Path  string   `json:"path"`
		Score int      `json:"score"`
		Rules []string `json:"rules"`
	}

	var results []fileResult
	failedCount := 0

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skip %s: %v\n", f, err)
			failedCount++
			continue
		}
		if !utf8.Valid(content) {
			fmt.Fprintf(os.Stderr, "Warning: skip %s: not valid UTF-8\n", f)
			failedCount++
			continue
		}
		s := strings.TrimSpace(string(content))
		if s == "" {
			fmt.Fprintf(os.Stderr, "Warning: skip %s: empty file\n", f)
			continue
		}
		q := prompt.ScorePromptQuality(s)
		rel, _ := filepath.Rel(cwd, f)
		if rel == "" || strings.HasPrefix(rel, "..") {
			rel = f
		}
		results = append(results, fileResult{Path: rel, Score: q.Score, Rules: q.Rules})
		if q.Score < minScore {
			failedCount++
		}
	}

	// If every file was skipped (unreadable/non-UTF-8), treat as failure
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No files could be scored.")
		if formatJSON {
			out := struct {
				Files    []fileResult `json:"files"`
				MinScore int          `json:"min_score"`
				OK       bool         `json:"ok"`
			}{nil, minScore, false}
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(out)
		}
		os.Exit(1)
	}

	if formatJSON {
		ok := failedCount == 0
		out := struct {
			Files    []fileResult `json:"files"`
			MinScore int          `json:"min_score"`
			OK       bool         `json:"ok"`
		}{results, minScore, ok}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(2)
		}
		if !ok {
			os.Exit(1)
		}
		return nil
	}

	for _, r := range results {
		rulesStr := strings.Join(r.Rules, ", ")
		if rulesStr != "" {
			rulesStr = "  (" + rulesStr + ")"
		}
		fmt.Printf("%s  %d%s\n", r.Path, r.Score, rulesStr)
	}

	if failedCount > 0 {
		os.Exit(1)
	}
	return nil
}
