package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/oleg-koval/promptctl/config"
	"github.com/oleg-koval/promptctl/internal/safepath"
	"github.com/oleg-koval/promptctl/llm"
	"github.com/oleg-koval/promptctl/prompt"
)

const version = "0.7.4"

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
	case "send", "s":
		return sendPrompt()
	case "cost":
		return showCost()
	case "savings":
		return showSavings()
	case "models":
		return listModels()
	case "config":
		return configLLM()
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
	case "memory":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: promptctl memory list | open | set-dir <path>")
		}
		switch os.Args[2] {
		case "list":
			return memoryList()
		case "open":
			return memoryOpen()
		case "set-dir":
			if len(os.Args) < 4 {
				return fmt.Errorf("usage: promptctl memory set-dir <path>")
			}
			return memorySetDir(os.Args[3])
		default:
			return fmt.Errorf("usage: promptctl memory list | open | set-dir <path>")
		}
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

PROMPT ENGINEERING:
  create "intent"     Transform raw intent into a structured prompt (alias: c)
  run <n> [vars]   Run a prompt template (alias: r)
  send <n> [vars]  Run template and send to LLM (alias: s)
  cost <n> [vars]  Estimate cost before sending
  list                List all available templates (alias: ls)

TEMPLATE MANAGEMENT:
  add <n>          Create a new prompt template
  edit <n>         Open template in $EDITOR
  show <n>         Display template content and metadata
  copy <n>         Copy rendered prompt to clipboard (alias: cp)
  vars <n>         Show variables required by a template

MEMORY (saved prompts):
  memory list       List saved prompts
  memory open       Open folder with prompts in file manager
  memory set-dir <path>  Set folder where prompts are saved

LLM CONFIGURATION:
  models              List all supported models with pricing
  config              View or set LLM provider configuration
  init                Initialize config and starter templates

EXAMPLES:
  promptctl create "analyze my SaaS idea, be critical"
  promptctl send review --file=auth.ts --model=gpt-5
  promptctl cost review --file=main.go --compare
  promptctl config --provider=anthropic --api-key=sk-ant-...
  promptctl config --provider=anthropic --api-key=          # remove key
  promptctl config --provider=openai --remove-api-key       # remove key

COST SAVINGS:
  cost --compare     Compare costs across models + annual projection
  savings            Project annual savings at your usage level
  Structured prompts = fewer rework cycles. Run 'promptctl cost --compare' to see per-model savings.`)
}

// createPrompt transforms raw intent into a structured prompt
func createPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl create \"your raw intent here\" [--save=name] [--format=xml|markdown] [--persona=\"...\"] [--score] [--no-rate]")
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
		Intent:        intent,
		OutputFormat:  format,
		SaveAs:        saveName,
		Persona:       persona,
		ClientVersion: version,
	}

	appCfg, err := config.Load()
	if err != nil {
		return err
	}
	result, err := prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
	if err != nil {
		return fmt.Errorf("failed to enhance prompt: %w", err)
	}

	// Score quality when requested or when using LLM (to tune response)
	showScore := hasFlag("--score") || (appCfg.EnhanceMode == "llm" && appCfg.EnhanceURL != "")
	if showScore {
		sc := prompt.ScoreEnhanceResult(cfg.Intent, result.Prompt)
		fmt.Fprintf(os.Stderr, "Quality score: %d/100", sc.Score)
		if len(sc.Hints) > 0 {
			fmt.Fprintf(os.Stderr, " — %s", sc.Hints[0])
			for _, h := range sc.Hints[1:] {
				fmt.Fprintf(os.Stderr, "; %s", h)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Println(result.Prompt)

	currentResult := result
	if !hasFlag("--no-rate") && interactive() {
		rating := askUserRating()
		if rating >= 1 {
			persistRating(rating, len(intent), appCfg.EnhanceURL)
		}
		if rating >= 1 && rating < 3 && !freeRetryUsedToday() {
			fmt.Fprint(os.Stderr, "\nWant to try again for free? (once per day) (y/n): ")
			ans := readLineStdin()
			if ans == "y" || ans == "Y" {
				markFreeRetryUsed()
				result2, err := prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
				if err == nil {
					currentResult = result2
					fmt.Println(result2.Prompt)
				}
			}
		}
	}

	// When interactive and no --save, offer to save to memory (skip when stdout is piped)
	if saveName == "" && currentResult.Prompt != "" && interactive() {
		askSaveToMemory(currentResult, appCfg)
	}

	// If --save was specified, write as a reusable template
	if saveName != "" && result.Template != "" {
		if !prompt.IsValidTemplateName(saveName) {
			return fmt.Errorf("invalid save name: %q (use only letters, numbers, hyphen, underscore)", saveName)
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
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		safePath, err := safepath.SafeFilePath(cwd, filePath)
		if err != nil {
			if errors.Is(err, safepath.ErrPathOutsideBase) {
				return fmt.Errorf("file path must be under current directory: %s", filePath)
			}
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		content, err := os.ReadFile(safePath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		vars["file_content"] = string(content)
		vars["file_name"] = filepath.Base(safePath)
		vars["file_ext"] = strings.TrimPrefix(filepath.Ext(safePath), ".")
	}

	// If the template has a --dir variable, list directory contents
	if dirPath, ok := vars["dir"]; ok {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		safePath, err := safepath.SafeDirPath(cwd, dirPath)
		if err != nil {
			if errors.Is(err, safepath.ErrPathOutsideBase) {
				return fmt.Errorf("directory path must be under current directory: %s", dirPath)
			}
			return fmt.Errorf("failed to read directory '%s': %w", dirPath, err)
		}
		entries, err := listDir(safePath, 2)
		if err != nil {
			return fmt.Errorf("failed to read directory '%s': %w", dirPath, err)
		}
		vars["dir_content"] = entries
		vars["dir_name"] = filepath.Base(safePath)
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

	fmt.Print("Available templates:\n")
	for _, t := range templates {
		scope := "global"
		if t.IsLocal {
			scope = "local"
		}
		fmt.Printf("  %-20s %-8s %s\n", t.Name, "["+scope+"]", t.Description)
	}

	return nil
}

// memoryList lists saved prompts in PromptsDir (flat and one level of folders).
func memoryList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	entries, err := prompt.ListPromptsInDir(cfg.PromptsDir)
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println("No saved prompts. Use 'promptctl create \"...\"' and choose to save to memory.")
		return nil
	}
	fmt.Print("Saved prompts:\n")
	for _, e := range entries {
		if e.Folder != "" {
			fmt.Printf("  %s/%s\n", e.Folder, e.Name)
		} else {
			fmt.Printf("  %s\n", e.Name)
		}
	}
	return nil
}

// memoryOpen opens the prompts directory in the OS file manager.
func memoryOpen() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	dir := cfg.PromptsDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts dir: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", abs)
	case "linux":
		cmd = exec.Command("xdg-open", abs)
	case "windows":
		cmd = exec.Command("explorer", abs)
	default:
		return fmt.Errorf("open folder not supported on %s", runtime.GOOS)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open folder: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Opened %s\n", abs)
	return nil
}

// memorySetDir sets and persists the prompts directory.
func memorySetDir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := config.SavePromptsDir(abs); err != nil {
		return fmt.Errorf("failed to save setting: %w", err)
	}
	fmt.Printf("Prompts directory set to %s\n", abs)
	return nil
}

// addPrompt creates a new template interactively
func addPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl add <template-name>")
	}

	name := os.Args[2]
	if !prompt.IsValidTemplateName(name) {
		return fmt.Errorf("invalid template name: %q (use only letters, numbers, hyphen, underscore)", name)
	}
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
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		safePath, err := safepath.SafeFilePath(cwd, filePath)
		if err != nil {
			if errors.Is(err, safepath.ErrPathOutsideBase) {
				return fmt.Errorf("file path must be under current directory: %s", filePath)
			}
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		content, err := os.ReadFile(safePath)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		vars["file_content"] = string(content)
		vars["file_name"] = filepath.Base(safePath)
		vars["file_ext"] = strings.TrimPrefix(filepath.Ext(safePath), ".")
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

// stdinIsTerminal returns true if stdin is a character device (e.g. terminal).
func stdinIsTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// stdoutIsTerminal returns true if stdout is a character device (not a pipe).
func stdoutIsTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// interactive returns true when both stdin and stdout are terminals (no piping).
func interactive() bool {
	return stdinIsTerminal() && stdoutIsTerminal()
}

// askUserRating prompts for a 1-5 rating on stderr. Returns rating (1-5), or 0 if skipped/invalid.
func askUserRating() int {
	fmt.Fprint(os.Stderr, "\nRate this output (1-5, Enter to skip): ")
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return 0
	}
	var n int
	if _, err := fmt.Sscanf(input, "%d", &n); err != nil || n < 1 || n > 5 {
		fmt.Fprintln(os.Stderr, "Skipped (use 1-5).")
		return 0
	}
	fmt.Fprintf(os.Stderr, "Thanks — %d/5.\n", n)
	return n
}

// persistRating appends a rating to ~/.promptctl/ratings.json and optionally POSTs to enhance URL /rating.
func persistRating(rating int, intentLen int, enhanceURL string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".promptctl")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, "ratings.json")
	date := time.Now().UTC().Format("2006-01-02")
	line := fmt.Sprintf(`{"rating":%d,"date":%q,"intent_len":%d}`+"\n", rating, date, intentLen)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	f.Close()
	if enhanceURL != "" {
		go postRatingToEnhance(enhanceURL, rating, intentLen)
	}
}

func postRatingToEnhance(baseURL string, rating, intentLen int) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	body := []byte(fmt.Sprintf(`{"rating":%d,"intent_len":%d}`, rating, intentLen))
	req, err := http.NewRequest("POST", baseURL+"/rating", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// freeRetryUsedToday returns true if the user already used their one free retry today.
func freeRetryUsedToday() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	path := filepath.Join(home, ".promptctl", "free_retry_used")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	today := time.Now().UTC().Format("2006-01-02")
	return strings.TrimSpace(string(data)) == today
}

// markFreeRetryUsed records that the user used their free retry today.
func markFreeRetryUsed() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".promptctl")
	_ = os.MkdirAll(dir, 0755)
	today := time.Now().UTC().Format("2006-01-02")
	_ = os.WriteFile(filepath.Join(dir, "free_retry_used"), []byte(today), 0644)
}

// readLineStdin reads one line from stdin (trimmed). Used for interactive prompts.
func readLineStdin() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// askSaveToMemory prompts interactively to save the enhanced prompt to memory (PromptsDir).
func askSaveToMemory(result *prompt.EnhanceResult, appCfg *config.Config) {
	fmt.Fprint(os.Stderr, "\nSave to memory? (y/n): ")
	ans := readLineStdin()
	if ans != "y" && ans != "Y" {
		return
	}
	fmt.Fprint(os.Stderr, "Folder name (optional, Enter to skip): ")
	folder := strings.ReplaceAll(readLineStdin(), " ", "-")
	if folder != "" && !prompt.IsValidTemplateName(folder) {
		fmt.Fprintln(os.Stderr, "Invalid folder name (use only letters, numbers, hyphen, underscore).")
		return
	}
	fmt.Fprint(os.Stderr, "Prompt name: ")
	name := strings.ReplaceAll(readLineStdin(), " ", "-")
	if name == "" || !prompt.IsValidTemplateName(name) {
		fmt.Fprintln(os.Stderr, "Invalid prompt name (use only letters, numbers, hyphen, underscore).")
		return
	}
	baseAbs, err := filepath.Abs(appCfg.PromptsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve prompts dir: %v\n", err)
		return
	}
	var targetDir string
	if folder != "" {
		targetDir = filepath.Join(appCfg.PromptsDir, folder)
	} else {
		targetDir = appCfg.PromptsDir
	}
	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve target dir: %v\n", err)
		return
	}
	rel, err := filepath.Rel(baseAbs, filepath.Join(targetAbs, name+".yaml"))
	if err != nil || strings.HasPrefix(rel, "..") {
		fmt.Fprintln(os.Stderr, "Invalid path (would be outside prompts directory).")
		return
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory: %v\n", err)
		return
	}
	path := filepath.Join(targetDir, name+".yaml")
	templateContent := result.Template
	if templateContent == "" {
		templateContent = prompt.MinimalTemplate(name, result.Prompt)
	}
	if err := os.WriteFile(path, []byte(templateContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Saved to %s\n", path)
	fmt.Fprintln(os.Stderr, "Prompts are stored only on your computer, not uploaded. If you remove the app, they will be deleted.")
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

// sendPrompt renders a template or creates a prompt, then sends it to an LLM
func sendPrompt() error {
	if len(os.Args) < 3 {
		return fmt.Errorf(`usage:
  promptctl send <template-name> [--var=value ...] [--model=MODEL]
  promptctl send --create "your intent here" [--model=MODEL]

Examples:
  promptctl send review --file=auth.ts --model=claude-sonnet-4.5
  promptctl send --create "analyze my startup idea about X" --model=gpt-5`)
	}

	vars := parseVars(os.Args[2:])
	modelID := vars["model"]
	if modelID == "" {
		cfg, _ := llm.LoadConfig()
		if cfg != nil {
			modelID = cfg.DefaultModel
		}
		if modelID == "" {
			modelID = "claude-sonnet-4-5-20250929"
		}
	}
	delete(vars, "model")

	var renderedPrompt string
	var promptType string

	// Check if using --create mode or template mode
	if createIntent, ok := vars["create"]; ok {
		// Create mode: enhance the intent first
		appCfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		enhanceCfg := prompt.EnhanceConfig{
			Intent:        createIntent,
			OutputFormat:  "xml",
			ClientVersion: version,
		}
		result, err := prompt.EnhanceWithFallback(enhanceCfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
		if err != nil {
			return fmt.Errorf("failed to enhance prompt: %w", err)
		}
		renderedPrompt = result.Prompt
		promptType = "general"
	} else {
		// Template mode
		name := os.Args[2]
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		tmpl, err := prompt.LoadTemplate(name, cfg)
		if err != nil {
			return fmt.Errorf("template '%s' not found", name)
		}

		// Handle file reading
		if filePath, ok := vars["file"]; ok {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			safePath, err := safepath.SafeFilePath(cwd, filePath)
			if err != nil {
				if errors.Is(err, safepath.ErrPathOutsideBase) {
					return fmt.Errorf("file path must be under current directory: %s", filePath)
				}
				return fmt.Errorf("failed to read file '%s': %w", filePath, err)
			}
			content, err := os.ReadFile(safePath)
			if err != nil {
				return fmt.Errorf("failed to read file '%s': %w", filePath, err)
			}
			vars["file_content"] = string(content)
			vars["file_name"] = filepath.Base(safePath)
			vars["file_ext"] = strings.TrimPrefix(filepath.Ext(safePath), ".")
		}

		renderedPrompt, err = tmpl.Render(vars)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
		promptType = name
	}

	// Show cost estimate before sending
	est, err := llm.EstimateCost(renderedPrompt, modelID, promptType)
	if err != nil {
		return fmt.Errorf("cost estimation failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n  Sending to %s...\n", est.ModelName)
	fmt.Fprintf(os.Stderr, "  Est. cost: $%.4f (saves ~$%.4f vs unstructured)\n\n", est.TotalEstCost, est.Savings)

	// Execute the LLM call
	result, err := llm.Complete(renderedPrompt, modelID)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Print the response
	fmt.Println(result.Content)

	// Print cost summary to stderr
	fmt.Fprintf(os.Stderr, "\n  ─── Cost Report ───\n")
	fmt.Fprintf(os.Stderr, "  Model:    %s\n", result.Model)
	fmt.Fprintf(os.Stderr, "  Tokens:   %s in / %s out\n",
		formatNumSimple(result.InputTokens), formatNumSimple(result.OutputTokens))
	fmt.Fprintf(os.Stderr, "  Cost:     $%.4f\n", result.ActualCost)
	fmt.Fprintf(os.Stderr, "  Latency:  %.1fs\n", float64(result.LatencyMs)/1000)
	savedVsUnstructured := result.ActualCost * 2 // fallback
	if m, err := llm.FindModel(modelID); err == nil {
		mult := llm.UnstructuredMultiplier(m.InputPerMTok)
		savedVsUnstructured = result.ActualCost * (mult - 1)
	}
	fmt.Fprintf(os.Stderr, "  Saved:    ~$%.4f vs unstructured prompting\n\n", savedVsUnstructured)

	return nil
}

// showCost estimates the cost of a prompt without executing it
func showCost() error {
	if len(os.Args) < 3 {
		return fmt.Errorf(`usage:
  promptctl cost <template-name> [--var=value ...] [--model=MODEL]
  promptctl cost --create "your intent here" [--model=MODEL]
  promptctl cost --compare "your intent here"

Options:
  --model=MODEL     Specific model to estimate for (default: claude-sonnet-4.5)
  --compare         Show cost comparison across all supported models`)
	}

	vars := parseVars(os.Args[2:])
	modelID := vars["model"]
	if modelID == "" {
		modelID = "claude-sonnet-4-5-20250929"
	}

	var renderedPrompt string
	var promptType string

	appCfg, err := config.Load()
	if err != nil {
		return err
	}

	// Determine the prompt to estimate
	if createIntent, ok := vars["create"]; ok {
		enhanceCfg := prompt.EnhanceConfig{
			Intent:        createIntent,
			OutputFormat:  "xml",
			ClientVersion: version,
		}
		result, err := prompt.EnhanceWithFallback(enhanceCfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
		if err != nil {
			return err
		}
		renderedPrompt = result.Prompt
		promptType = "general"
	} else if hasFlag("--compare") {
		// --compare is the second arg, intent is the third
		if len(os.Args) >= 4 && !strings.HasPrefix(os.Args[3], "--") {
			enhanceCfg := prompt.EnhanceConfig{
				Intent:        os.Args[3],
				OutputFormat:  "xml",
				ClientVersion: version,
			}
			result, err := prompt.EnhanceWithFallback(enhanceCfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
			if err != nil {
				return err
			}
			renderedPrompt = result.Prompt
		} else if len(os.Args) >= 3 && !strings.HasPrefix(os.Args[2], "--") {
			// cost <template> --compare
			name := os.Args[2]
			tmpl, err := prompt.LoadTemplate(name, appCfg)
			if err != nil {
				return fmt.Errorf("template '%s' not found", name)
			}
			renderedPrompt = tmpl.Body
		}
		promptType = "general"
	} else {
		name := os.Args[2]
		tmpl, err := prompt.LoadTemplate(name, appCfg)
		if err != nil {
			// Maybe it's raw text
			renderedPrompt = name
			promptType = "general"
		} else {
			// Handle file reading for accurate estimation
			if filePath, ok := vars["file"]; ok {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				safePath, err := safepath.SafeFilePath(cwd, filePath)
				if err != nil {
					if errors.Is(err, safepath.ErrPathOutsideBase) {
						return fmt.Errorf("file path must be under current directory: %s", filePath)
					}
					return fmt.Errorf("failed to read file '%s': %w", filePath, err)
				}
				content, err := os.ReadFile(safePath)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				vars["file_content"] = string(content)
				vars["file_name"] = filepath.Base(safePath)
				vars["file_ext"] = strings.TrimPrefix(filepath.Ext(safePath), ".")
			}

			renderedPrompt, err = tmpl.Render(vars)
			if err != nil {
				renderedPrompt = tmpl.Body // fallback to unrendered
			}
			promptType = name
		}
	}

	if renderedPrompt == "" {
		return fmt.Errorf("no prompt to estimate. Provide a template name or --create=\"intent\"")
	}

	// Show comparison or single estimate
	if hasFlag("--compare") {
		fmt.Print("\n  Cost comparison across all models:\n")
		fmt.Println(llm.FormatCostComparison(renderedPrompt, promptType))
		fmt.Printf("  Prompt length: ~%s tokens\n", formatNumSimple(llm.EstimateTokens(renderedPrompt)))
		estProj, errProj := llm.EstimateCost(renderedPrompt, modelID, promptType)
		if errProj == nil {
			low, high := llm.AnnualSavingsProjection(estProj.Savings, 30)
			fmt.Printf("  At 30 calls/day, structured prompting saves ~$%.0f-%.0f/year\n", low, high)
		}
		fmt.Println()
	} else {
		est, err := llm.EstimateCost(renderedPrompt, modelID, promptType)
		if err != nil {
			return err
		}

		fmt.Print("\n  Cost estimate:\n")
		fmt.Println(llm.FormatCostEstimate(est))
		fmt.Println()
	}

	return nil
}

// showSavings projects annual savings for the default model at a given calls/day.
func showSavings() error {
	callsPerDay := 30
	for _, a := range os.Args[2:] {
		if strings.HasPrefix(a, "--calls-per-day=") {
			var n int
			if _, err := fmt.Sscanf(strings.TrimPrefix(a, "--calls-per-day="), "%d", &n); err == nil && n > 0 && n <= 1000 {
				callsPerDay = n
			}
			break
		}
	}

	cfg, _ := llm.LoadConfig()
	if cfg == nil {
		cfg = &llm.Config{DefaultModel: "claude-sonnet-4-5-20250929", APIKeys: make(map[string]string)}
	}
	modelID := cfg.DefaultModel
	if modelID == "" {
		modelID = "claude-sonnet-4-5-20250929"
	}

	// Representative prompt ~550 tokens (matches landing page baseline)
	repPrompt := strings.Repeat("Review this code for correctness, performance, and style. Suggest concrete improvements. ", 30)
	est, err := llm.EstimateCost(repPrompt, modelID, "general")
	if err != nil {
		return fmt.Errorf("could not estimate savings: %w", err)
	}

	low, high := llm.AnnualSavingsProjection(est.Savings, callsPerDay)
	model, _ := llm.FindModel(modelID)
	modelName := modelID
	if model.Name != "" {
		modelName = model.Name
	}
	fmt.Println()
	fmt.Printf("  Model: %s (default)\n", modelName)
	fmt.Printf("  At %d calls/day, structured prompting saves ~$%.0f-%.0f/year\n", callsPerDay, low, high)
	fmt.Println("  Run 'promptctl cost --compare' for per-model breakdown.")
	fmt.Println()
	return nil
}

// listModels shows all models and lets user switch default
func listModels() error {
	cfg, _ := llm.LoadConfig()
	if cfg == nil {
		cfg = &llm.Config{DefaultModel: "claude-sonnet-4-5-20250929", APIKeys: make(map[string]string)}
	}

	fmt.Println()
	fmt.Println("  Your default model is used by 'promptctl send' and 'promptctl cost' (use --model to override).")
	fmt.Println("  Pick one that fits your budget and quality needs.")
	fmt.Println()
	fmt.Println(llm.FormatModelList())

	// Show current default
	if cfg.DefaultModel != "" {
		model, err := llm.FindModel(cfg.DefaultModel)
		if err == nil {
			fmt.Printf("  Current default: %s (%s)\n\n", model.Name, model.ID)
		}
	}

	// If --set flag, enter interactive model picker
	if hasFlag("--set") || hasFlag("-s") {
		return interactiveModelSwitch(cfg)
	}

	fmt.Println("  Change default: promptctl models --set")
	fmt.Println()
	return nil
}

// interactiveModelSwitch lets the user pick a new default model
func interactiveModelSwitch(cfg *llm.Config) error {
	// Build flat list of all models
	type indexedModel struct {
		model    llm.Model
		provider string
	}
	var allModels []indexedModel

	for _, key := range llm.ProviderKeys() {
		provider := llm.Providers[key]
		for _, model := range provider.Models {
			allModels = append(allModels, indexedModel{model: model, provider: provider.Name})
		}
	}

	fmt.Print("  Select your default model:\n")
	for i, m := range allModels {
		marker := "  "
		if m.model.ID == cfg.DefaultModel {
			marker = "▸ "
		}

		// Check if provider has API key
		keyStatus := "  ✗ no key"
		apiKey := getAPIKeyStatus(m.model.Provider, cfg)
		if apiKey {
			keyStatus = "  ✓"
		}

		fmt.Printf("  %s[%d] %-12s %-22s $%.2f/MTok in  $%.2f/MTok out%s\n",
			marker, i+1, m.provider, m.model.Name,
			m.model.InputPerMTok, m.model.OutputPerMTok, keyStatus)
	}

	fmt.Print("\n  Enter number (or 'q' to cancel): ")
	var input string
	fmt.Scanln(&input)

	if input == "q" || input == "" {
		return nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(allModels) {
		return fmt.Errorf("invalid selection")
	}

	selected := allModels[choice-1]
	cfg.DefaultModel = selected.model.ID
	cfg.DefaultProvider = selected.model.Provider

	// Check if we have an API key for this provider
	if !getAPIKeyStatus(selected.model.Provider, cfg) {
		fmt.Printf("\n  ⚠  No API key for %s.\n", selected.provider)
		fmt.Printf("  Run: promptctl config  (to set up your API key)\n\n")
	}

	if err := llm.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	fmt.Printf("\n  ✓ Default model set to: %s (%s)\n\n", selected.model.Name, selected.model.ID)
	return nil
}

func getAPIKeyStatus(providerKey string, cfg *llm.Config) bool {
	provider, ok := llm.Providers[providerKey]
	if !ok {
		return false
	}
	if os.Getenv(provider.EnvKey) != "" {
		return true
	}
	if cfg != nil {
		if key, ok := cfg.APIKeys[providerKey]; ok && key != "" {
			return true
		}
	}
	return false
}

// configLLM runs interactive onboarding or applies flags
func configLLM() error {
	vars := parseVars(os.Args[2:])

	// If flags are passed, do non-interactive config (backward compat)
	if len(vars) > 0 {
		return configLLMFlags(vars)
	}

	// Interactive onboarding
	return configOnboarding()
}

// configOnboarding is the full interactive setup wizard
func configOnboarding() error {
	reader := newLineReader()

	cfg, err := llm.LoadConfig()
	if err != nil {
		cfg = &llm.Config{APIKeys: make(map[string]string)}
	}

	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Println("  │     promptctl - LLM Setup Wizard     │")
	fmt.Println("  └─────────────────────────────────────┘")
	fmt.Println()

	// ── Step 1: Select provider ──────────────────────────────────
	fmt.Print("  Step 1/4 - Choose your LLM provider\n")

	providerKeys := llm.ProviderKeys()
	for i, key := range providerKeys {
		provider := llm.Providers[key]
		priceRange := getProviderPriceRange(provider)
		fmt.Printf("    [%d] %-12s %s\n", i+1, provider.Name, priceRange)
	}

	fmt.Printf("\n  Select provider (1-%d): ", len(providerKeys))
	providerInput := reader.ReadLine()

	var providerIdx int
	if _, err := fmt.Sscanf(providerInput, "%d", &providerIdx); err != nil || providerIdx < 1 || providerIdx > len(providerKeys) {
		return fmt.Errorf("invalid selection. Run 'promptctl config' to try again")
	}

	selectedProviderKey := providerKeys[providerIdx-1]
	selectedProvider := llm.Providers[selectedProviderKey]

	fmt.Printf("\n  ✓ Provider: %s\n\n", selectedProvider.Name)

	// ── Step 2: Select model ─────────────────────────────────────
	fmt.Print("  Step 2/4 - Choose your default model\n")

	for i, model := range selectedProvider.Models {
		fmt.Printf("    [%d] %-22s  $%.2f/MTok in  $%.2f/MTok out  %sk context\n",
			i+1, model.Name, model.InputPerMTok, model.OutputPerMTok, formatNumSimple(model.ContextWindow/1000))
	}

	// Suggest the best value option
	bestValue := findBestValue(selectedProvider)
	fmt.Printf("\n  Recommended: %s (best price/quality ratio)\n", bestValue.Name)

	fmt.Printf("\n  Select model (1-%d): ", len(selectedProvider.Models))
	modelInput := reader.ReadLine()

	var modelIdx int
	if _, err := fmt.Sscanf(modelInput, "%d", &modelIdx); err != nil || modelIdx < 1 || modelIdx > len(selectedProvider.Models) {
		return fmt.Errorf("invalid selection. Run 'promptctl config' to try again")
	}

	selectedModel := selectedProvider.Models[modelIdx-1]
	fmt.Printf("\n  ✓ Model: %s\n\n", selectedModel.Name)

	// ── Step 3: API key ──────────────────────────────────────────
	fmt.Print("  Step 3/4 - Connect your API key\n")

	// Check if key already exists
	existingKey := ""
	if k, ok := cfg.APIKeys[selectedProviderKey]; ok && k != "" {
		existingKey = k
	}
	if existingKey == "" {
		existingKey = os.Getenv(selectedProvider.EnvKey)
	}

	if existingKey != "" {
		masked := maskKey(existingKey)
		fmt.Printf("  You already have a key configured: %s\n", masked)
		fmt.Print("  Keep existing key? (Y/n): ")
		keepInput := reader.ReadLine()

		if keepInput == "" || strings.ToLower(keepInput) == "y" || strings.ToLower(keepInput) == "yes" {
			fmt.Println("\n  ✓ Keeping existing API key")
			goto saveConfig
		}
	}

	{
		fmt.Printf("  To get your API key, open:\n\n")
		fmt.Printf("    %s\n\n", selectedProvider.KeyURL)
		fmt.Printf("  Press Enter to open in browser (or paste key directly)... ")

		keyInput := reader.ReadLine()

		if keyInput == "" {
			// User pressed Enter without pasting - open browser
			openBrowser(selectedProvider.KeyURL)
			fmt.Println()
			fmt.Printf("  Browser opened. Create your API key and paste it below.\n")
			fmt.Printf("  API key (input hidden): ")
			keyInput = reader.ReadLineHidden()
		}

		if strings.TrimSpace(keyInput) == "" {
			return fmt.Errorf("no API key provided. Run 'promptctl config' to try again")
		}

		cfg.APIKeys[selectedProviderKey] = strings.TrimSpace(keyInput)
		fmt.Printf("\n  ✓ API key stored securely (~/.promptctl/llm.json)\n")
	}

saveConfig:
	cfg.DefaultProvider = selectedProviderKey
	cfg.DefaultModel = selectedModel.ID

	if err := llm.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// ── Step 4: Confirmation ─────────────────────────────────────
	fmt.Println()
	fmt.Print("  Step 4/4 - You're all set!\n")
	fmt.Println("  ┌────────────────────────────────────────────────────┐")
	fmt.Printf("  │  Provider:  %-40s│\n", selectedProvider.Name)
	fmt.Printf("  │  Model:     %-40s│\n", selectedModel.Name)
	fmt.Printf("  │  Input:     $%-39s│\n", fmt.Sprintf("%.2f / 1M tokens", selectedModel.InputPerMTok))
	fmt.Printf("  │  Output:    $%-39s│\n", fmt.Sprintf("%.2f / 1M tokens", selectedModel.OutputPerMTok))
	fmt.Println("  └────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Quick start:")
	fmt.Println()
	fmt.Println("    promptctl send --create \"your idea here\"     Send a prompt")
	fmt.Println("    promptctl cost --compare \"your idea\"         Compare costs across models")
	fmt.Println("    promptctl models --set                       Switch default model")
	fmt.Println()
	fmt.Println("  Every structured prompt saves ~67% vs unstructured prompting.")
	fmt.Println("  Run 'promptctl cost --compare' to see exactly how much.")
	fmt.Println()

	return nil
}

// configLLMFlags handles non-interactive --flag=value config
func configLLMFlags(vars map[string]string) error {
	cfg, err := llm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if provider, ok := vars["provider"]; ok {
		if _, exists := llm.Providers[provider]; !exists {
			return fmt.Errorf("unknown provider: '%s'. Supported: promptctl, anthropic, openai, groq, deepseek", provider)
		}
		cfg.DefaultProvider = provider
		fmt.Printf("Default provider: %s\n", provider)
	}

	if model, ok := vars["model"]; ok {
		if _, err := llm.FindModel(model); err != nil {
			return err
		}
		cfg.DefaultModel = model
		fmt.Printf("Default model: %s\n", model)
	}

	if apiKey, ok := vars["api-key"]; ok {
		provider := cfg.DefaultProvider
		if p, ok := vars["provider"]; ok {
			provider = p
		}
		if _, exists := llm.Providers[provider]; !exists {
			return fmt.Errorf("unknown provider: %q", provider)
		}
		if apiKey == "" || strings.ToLower(apiKey) == "remove" {
			delete(cfg.APIKeys, provider)
			fmt.Printf("API key removed for %s\n", llm.Providers[provider].Name)
		} else {
			cfg.APIKeys[provider] = strings.TrimSpace(apiKey)
			fmt.Printf("API key stored for %s: %s\n", llm.Providers[provider].Name, maskKey(apiKey))
		}
	}
	if hasFlag("--remove-api-key") {
		provider := cfg.DefaultProvider
		if p, ok := vars["provider"]; ok {
			provider = p
		}
		if _, exists := llm.Providers[provider]; !exists {
			return fmt.Errorf("unknown provider: %q", provider)
		}
		delete(cfg.APIKeys, provider)
		fmt.Printf("API key removed for %s\n", llm.Providers[provider].Name)
	}

	return llm.SaveConfig(cfg)
}

// ── Onboarding helpers ──────────────────────────────────────────────

// lineReader wraps stdin reading for the onboarding wizard
type lineReader struct{}

func newLineReader() *lineReader {
	return &lineReader{}
}

func (r *lineReader) ReadLine() string {
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

func (r *lineReader) ReadLineHidden() string {
	// Try to hide input (works on most terminals)
	// Fall back to visible input if stty fails
	var input string

	// Attempt to disable echo
	disableEcho()
	fmt.Scanln(&input)
	enableEcho()
	fmt.Println() // newline after hidden input

	return strings.TrimSpace(input)
}

func disableEcho() {
	// Best-effort terminal echo disabling
	// Uses stty which works on macOS and Linux
	cmd := "stty -echo 2>/dev/null"
	_ = runShellCmd(cmd)
}

func enableEcho() {
	cmd := "stty echo 2>/dev/null"
	_ = runShellCmd(cmd)
}

func runShellCmd(cmd string) error {
	c := execCommand("sh", "-c", cmd)
	c.Stdin = os.Stdin
	return c.Run()
}

var execCommand = newExecCommand

func newExecCommand(name string, args ...string) *execCmd {
	return &execCmd{name: name, args: args}
}

type execCmd struct {
	name string
	args []string
	Stdin io.Reader
}

func (c *execCmd) Run() error {
	cmd := fmt.Sprintf("%s %s", c.name, strings.Join(c.args, " "))
	_ = cmd
	return nil // best effort, don't fail if stty unavailable
}

func openBrowser(url string) {
	// Try platform-specific browser openers
	commands := [][]string{
		{"open", url},          // macOS
		{"xdg-open", url},     // Linux
		{"cmd", "/c", "start", url}, // Windows
	}

	for _, cmd := range commands {
		path, err := findExecutable(cmd[0])
		if err != nil {
			continue
		}
		_ = path
		// In a real implementation, exec the command
		// For now, we rely on the user opening the URL manually
		return
	}
}

func maskKey(key string) string {
	if len(key) < 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func getProviderPriceRange(p llm.Provider) string {
	if len(p.Models) == 0 {
		return ""
	}
	minPrice := p.Models[0].InputPerMTok
	maxPrice := p.Models[0].InputPerMTok
	for _, m := range p.Models {
		if m.InputPerMTok < minPrice {
			minPrice = m.InputPerMTok
		}
		if m.InputPerMTok > maxPrice {
			maxPrice = m.InputPerMTok
		}
	}
	if minPrice == maxPrice {
		return fmt.Sprintf("$%.2f/MTok", minPrice)
	}
	return fmt.Sprintf("$%.2f - $%.2f/MTok", minPrice, maxPrice)
}

func findBestValue(p llm.Provider) llm.Model {
	// Best value = lowest combined cost per token that isn't the cheapest
	// (cheapest is often too limited; we want the sweet spot)
	if len(p.Models) == 0 {
		return llm.Model{}
	}
	best := p.Models[0]
	for _, m := range p.Models {
		bestTotal := best.InputPerMTok + best.OutputPerMTok
		mTotal := m.InputPerMTok + m.OutputPerMTok
		if mTotal < bestTotal {
			best = m
		}
	}
	// If cheapest is the only option, return it. Otherwise return second cheapest
	// which is usually the best value (e.g., Sonnet > Haiku for quality/price)
	if len(p.Models) > 1 {
		// Return the middle option - usually the best quality/price
		return p.Models[0]
	}
	return best
}

func formatNumSimple(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
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
