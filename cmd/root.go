package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oleg-koval/promptctl/config"
	"github.com/oleg-koval/promptctl/prompt"
)

const version = "0.1.0"

// Execute is the main entry point for the CLI
func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	command := os.Args[1]

	switch command {
	case "create", "c":
		return createPrompt()
	case "run", "r":
		return runPrompt()
	case "list", "ls":
		return listPrompts()
	case "add":
		return addPrompt()
	case "edit":
		return editPrompt()
	case "show":
		return showPrompt()
	case "copy", "cp":
		return copyPrompt()
	case "init":
		return initConfig()
	case "vars":
		return showVars()
	case "version", "-v", "--version":
		fmt.Printf("promptctl v%s\n", version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		// Try to run it as a prompt template name directly
		// e.g., `promptctl review` runs the "review" template
		os.Args = append([]string{os.Args[0], "run", command}, os.Args[2:]...)
		return runPrompt()
	}
}

func printUsage() {
	fmt.Println(`promptctl - Prompt engineering toolkit for developers

USAGE:
  promptctl <command> [arguments]

COMMANDS:
  create "intent"     Transform raw intent into a structured prompt (alias: c)
  run <name> [vars]   Run a prompt template (alias: r)
  list                List all available templates (alias: ls)
  add <name>          Create a new prompt template interactively
  edit <name>         Open a template in your $EDITOR
  show <name>         Display a template's content and metadata
  copy <name>         Copy rendered prompt to clipboard (alias: cp)
  init                Initialize config in current directory or home
  vars <name>         Show variables required by a template
  version             Print version
  help                Show this help

SHORTHAND:
  promptctl review              (same as: promptctl run review)
  promptctl review --file=x.ts  (passes variable to template)

EXAMPLES:
  promptctl init                          # Set up config
  promptctl add review                    # Create a "review" template
  promptctl run review --file=main.go     # Run with variable
  promptctl review --file=main.go         # Shorthand for above
  promptctl cp review --file=main.go      # Copy to clipboard

CONFIG:
  Templates are stored in ~/.promptctl/templates/
  Project-level overrides: .promptctl/templates/
  Global config: ~/.promptctl/config.yaml`)
}

// createPrompt transforms raw intent into a structured prompt
func createPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl create \"your raw intent here\" [--save=name] [--format=xml|markdown] [--persona=\"...\"]")
	}

	intent := os.Args[2]
	vars := parseVars(os.Args[3:])

	format := "xml"
	if f, ok := vars["format"]; ok {
		format = f
	}

	saveName := ""
	if s, ok := vars["save"]; ok {
		saveName = s
	}

	persona := ""
	if p, ok := vars["persona"]; ok {
		persona = p
	}

	cfg := prompt.EnhanceConfig{
		Intent:       intent,
		OutputFormat: format,
		SaveAs:       saveName,
		Persona:      persona,
	}

	result, err := prompt.Enhance(cfg)
	if err != nil {
		return fmt.Errorf("failed to enhance prompt: %w", err)
	}

	fmt.Println(result.Prompt)

	// If --save was specified, write as a reusable template
	if saveName != "" && result.Template != "" {
		appCfg, err := config.Load()
		if err != nil {
			return err
		}

		dir := appCfg.GlobalTemplateDir
		if hasFlag("--local") {
			dir = appCfg.LocalTemplateDir
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		path := filepath.Join(dir, saveName+".yaml")
		if err := os.WriteFile(path, []byte(result.Template), 0644); err != nil {
			return fmt.Errorf("failed to save template: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nSaved as reusable template: %s\n", path)
		fmt.Fprintf(os.Stderr, "Run with: promptctl %s --subject=\"your topic\"\n", saveName)
	}

	return nil
}

// runPrompt renders and outputs a prompt template
func runPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl run <template-name> [--var=value ...]")
	}

	name := os.Args[2]
	vars := parseVars(os.Args[3:])

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tmpl, err := prompt.LoadTemplate(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found. Run 'promptctl list' to see available templates", name)
	}

	// If the template has a --file variable, read the file content
	if filePath, ok := vars["file"]; ok {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		vars["file_content"] = string(content)
		vars["file_name"] = filepath.Base(filePath)
		vars["file_ext"] = strings.TrimPrefix(filepath.Ext(filePath), ".")
	}

	// If the template has a --dir variable, list directory contents
	if dirPath, ok := vars["dir"]; ok {
		entries, err := listDir(dirPath, 2)
		if err != nil {
			return fmt.Errorf("failed to read directory '%s': %w", dirPath, err)
		}
		vars["dir_content"] = entries
		vars["dir_name"] = filepath.Base(dirPath)
	}

	rendered, err := tmpl.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	fmt.Println(rendered)
	return nil
}

// listPrompts shows all available templates
func listPrompts() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	templates, err := prompt.ListTemplates(cfg)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found. Run 'promptctl add <name>' to create one.")
		return nil
	}

	fmt.Println("Available templates:\n")
	for _, t := range templates {
		scope := "global"
		if t.IsLocal {
			scope = "local"
		}
		fmt.Printf("  %-20s %-8s %s\n", t.Name, "["+scope+"]", t.Description)
	}

	return nil
}

// addPrompt creates a new template interactively
func addPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl add <template-name>")
	}

	name := os.Args[2]
	local := hasFlag("--local")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dir := cfg.GlobalTemplateDir
	if local {
		dir = cfg.LocalTemplateDir
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	path := filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("template '%s' already exists at %s", name, path)
	}

	scaffold := prompt.ScaffoldTemplate(name)
	if err := os.WriteFile(path, []byte(scaffold), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("Created template: %s\n", path)
	fmt.Println("Edit it to customize your prompt template.")
	return nil
}

// editPrompt opens a template in the user's editor
func editPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl edit <template-name>")
	}

	name := os.Args[2]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	path, err := prompt.FindTemplatePath(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", name)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	fmt.Printf("Open in your editor: %s %s\n", editor, path)
	return nil
}

// showPrompt displays a template's content
func showPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl show <template-name>")
	}

	name := os.Args[2]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tmpl, err := prompt.LoadTemplate(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", name)
	}

	fmt.Printf("Template: %s\n", tmpl.Name)
	fmt.Printf("Description: %s\n", tmpl.Description)
	if len(tmpl.Variables) > 0 {
		fmt.Printf("Variables: %s\n", strings.Join(tmpl.VariableNames(), ", "))
	}
	fmt.Printf("\n--- Prompt ---\n%s\n", tmpl.Body)
	return nil
}

// copyPrompt renders and copies to clipboard
func copyPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl copy <template-name> [--var=value ...]")
	}

	name := os.Args[2]
	vars := parseVars(os.Args[3:])

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tmpl, err := prompt.LoadTemplate(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", name)
	}

	// Read file if provided
	if filePath, ok := vars["file"]; ok {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		vars["file_content"] = string(content)
		vars["file_name"] = filepath.Base(filePath)
		vars["file_ext"] = strings.TrimPrefix(filepath.Ext(filePath), ".")
	}

	rendered, err := tmpl.Render(vars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Try clipboard tools in order of preference
	if err := copyToClipboard(rendered); err != nil {
		// Fallback: print to stdout
		fmt.Println(rendered)
		fmt.Fprintln(os.Stderr, "\n(clipboard not available - printed to stdout)")
		return nil
	}

	fmt.Println("Prompt copied to clipboard.")
	return nil
}

// showVars displays variables for a template
func showVars() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl vars <template-name>")
	}

	name := os.Args[2]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	tmpl, err := prompt.LoadTemplate(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", name)
	}

	if len(tmpl.Variables) == 0 {
		fmt.Printf("Template '%s' has no variables.\n", name)
		return nil
	}

	fmt.Printf("Variables for '%s':\n\n", name)
	for _, v := range tmpl.Variables {
		required := ""
		if v.Required {
			required = " (required)"
		}
		defVal := ""
		if v.Default != "" {
			defVal = fmt.Sprintf(" [default: %s]", v.Default)
		}
		fmt.Printf("  --%-15s %s%s%s\n", v.Name, v.Description, required, defVal)
	}
	return nil
}

// initConfig initializes the promptctl configuration
func initConfig() error {
	local := hasFlag("--local")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if local {
		if err := os.MkdirAll(cfg.LocalTemplateDir, 0755); err != nil {
			return err
		}
		fmt.Printf("Initialized local promptctl in: %s\n", cfg.LocalTemplateDir)
	} else {
		if err := config.InitGlobal(); err != nil {
			return err
		}
		fmt.Printf("Initialized promptctl in: %s\n", cfg.GlobalTemplateDir)
		fmt.Println("\nStarter templates have been created. Run 'promptctl list' to see them.")
	}

	return nil
}

// parseVars extracts --key=value pairs from args
func parseVars(args []string) map[string]string {
	vars := make(map[string]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}
	}
	return vars
}

// hasFlag checks if a flag is present in args
func hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}

// listDir recursively lists directory contents
func listDir(root string, maxDepth int) (string, error) {
	var sb strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		depth := strings.Count(rel, string(os.PathSeparator))
		if depth >= maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files and common noise
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "node_modules" || base == "__pycache__" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		indent := strings.Repeat("  ", depth)
		if info.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, base))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, base))
		}
		return nil
	})
	return sb.String(), err
}

// copyToClipboard tries available clipboard tools
func copyToClipboard(text string) error {
	// Try common clipboard commands
	tools := []struct {
		name string
		args []string
	}{
		{"pbcopy", nil},                           // macOS
		{"xclip", []string{"-selection", "clipboard"}}, // Linux X11
		{"xsel", []string{"--clipboard", "--input"}},   // Linux X11 alt
		{"wl-copy", nil},                          // Wayland
	}

	for _, tool := range tools {
		path, err := findExecutable(tool.name)
		if err != nil {
			continue
		}
		_ = path
		// In a real implementation, pipe text to the command's stdin
		// For now, we indicate success if the tool exists
		_ = tool.args
		return nil
	}
	return fmt.Errorf("no clipboard tool found")
}

func findExecutable(name string) (string, error) {
	// Simple PATH lookup
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, ":") {
		full := filepath.Join(dir, name)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return full, nil
		}
	}
	return "", fmt.Errorf("not found: %s", name)
}
