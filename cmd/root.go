package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prompt-ctl/promptctl/config"
	"github.com/prompt-ctl/promptctl/internal/cloud"
	"github.com/prompt-ctl/promptctl/internal/onboarding"
	"github.com/prompt-ctl/promptctl/internal/safepath"
	"github.com/prompt-ctl/promptctl/internal/shell"
	"github.com/prompt-ctl/promptctl/internal/ui"
	"github.com/prompt-ctl/promptctl/llm"
	"github.com/prompt-ctl/promptctl/prompt"
)

const version = "1.0.0"

const githubReleasesLatest = "https://api.github.com/repos/prompt-ctl/promptctl/releases/latest"
const autoUpdateInterval = 24 * time.Hour

// Execute is the main entry point for the CLI
func Execute() error {
	// First-time setup: run onboarding once (init + LLM config + aliases) then continue.
	if interactive() && !onboarding.FirstRunDone() {
		runFirstTimeOnboarding()
		_ = onboarding.MarkFirstRunDone()
	}

	if len(os.Args) < 2 {
		printUsage()
		maybeShowAliasTip()
		return nil
	}

	command := os.Args[1]
	if msg := autoUpdateOnLaunch(command); msg != "" {
		fmt.Fprintln(os.Stderr, ui.Hint("↑ "+msg))
	}

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
	case "score":
		return runScore()
	case "fix":
		return runFix()
	case "experiment", "exp":

		if len(os.Args) > 2 && os.Args[2] == "optimize" {
			return runOptimize()
		}

		return runExperiment()
	case "test", "t":
		return runTest()
	case "version", "-v", "--version":
		fmt.Printf("promptctl v%s\n", version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		maybeShowAliasTip()
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
  New? Run 'promptctl init' to set up; then 'promptctl savings' to see your potential annual savings.
  Shortcuts: Run 'promptctl init' to add shell aliases (prompt, p) — then use: prompt create "..." or p list.

PROMPT ENGINEERING:
  create "intent"     Transform raw intent into a structured prompt (alias: c)
  run <n> [vars]   Run a prompt template (alias: r)
  send <n> [vars]  Run template and send to LLM (alias: s)
  cost <n> [vars]  Estimate cost before sending
  test <n> [vars]  Run prompt tests (regression + model compare) (alias: t)
  experiment <n> [vars]  Benchmark template across models (alias: exp)
  experiment optimize <n>  Generate prompt variants and keep best
  list                List all available templates (alias: ls)

TEMPLATE MANAGEMENT:
  add <n>          Create a new prompt template
  edit <n>         Open template in $EDITOR
  show <n>         Display template content and metadata
  copy <n>         Copy rendered prompt to clipboard (alias: cp)
  vars <n>         Show variables required by a template

MEMORY (saved prompts):
  Prompts you saved from create.
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
		return fmt.Errorf("usage: promptctl create \"your raw intent here\" [--save=name] [--format=markdown|xml|yaml|json|text] [--persona=\"...\"] [--score] [--no-rate]")
	}

	intent := os.Args[2]
	vars := parseVars(os.Args[3:])

	appCfg, err := config.Load()
	if err != nil {
		return err
	}
	cloudClient := cloud.New(appCfg.CloudEnabled, appCfg.CloudBaseURL)

	format := "markdown"
	if f, ok := vars["format"]; ok {
		format = f
	} else if appCfg.DefaultCreateFormat != "" {
		format = appCfg.DefaultCreateFormat
	} else if interactive() {
		var formatChoice string
		formatOptions := []string{"Markdown (recommended)", "XML", "YAML", "JSON", "Plain text"}
		if err := ui.SelectOption("Output format", formatOptions, &formatChoice); err == nil {
			switch formatChoice {
			case "Markdown (recommended)":
				format = "markdown"
			case "XML":
				format = "xml"
			case "YAML", "JSON", "Plain text":
				format = "markdown"
			default:
				format = "markdown"
			}
			remember, _ := ui.Confirm("Remember my choice?", true)
			if remember {
				_ = config.SaveCreateFormat(format)
			}
			fmt.Fprintln(os.Stderr, ui.Hint("To change output format later: promptctl create --format="+format+" \"...\" (use markdown, xml, yaml, json, or text)"))
		}
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

	showSpinner := appCfg.EnhanceMode == "llm" && stderrIsTerminal()
	spinnerModel := ""
	var rawTokenEst int
	if showSpinner {
		rawTokenEst = llm.EstimateTokens(intent)
		if llmCfg, _ := llm.LoadConfig(); llmCfg != nil && llmCfg.DefaultModel != "" {
			if m, err := llm.FindModel(llmCfg.DefaultModel); err == nil {
				spinnerModel = m.Name
			}
		}
	}
	var result *prompt.EnhanceResult
	if showSpinner {
		done := make(chan struct{})
		go runSpinner(done, spinnerModel, rawTokenEst)
		result, err = prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
		close(done)
	} else {
		result, err = prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
	}
	if err != nil {
		return fmt.Errorf("failed to enhance prompt: %w", err)
	}
	// Score quality when requested or when using LLM (to tune response)
	showScore := hasFlag("--score") || (appCfg.EnhanceMode == "llm" && appCfg.EnhanceURL != "")
	if showScore {
		sc := prompt.ScoreEnhanceResult(cfg.Intent, result.Prompt)
		printQualityScoreBox(sc.Score, sc.Hints)
	}

	fmt.Println(ui.FormatPromptForTerminal(result.Prompt))

	if interactive() {
		copyIt, err := ui.Confirm("\n  Copy prompt to clipboard?", true)
		if err == nil && copyIt {
			if err := copyToClipboard(result.Prompt); err != nil {
				fmt.Fprintln(os.Stderr, ui.Hint("  (clipboard unavailable: "+err.Error()+")"))
			} else {
				fmt.Fprintln(os.Stderr, ui.Success("  ✓ Copied to clipboard."))
			}
		}
	}

	currentResult := result
	if !hasFlag("--no-rate") && interactive() {
		const maxTries = 15
		for try := 1; try <= maxTries; try++ {
			rating := askUserRating()
			if rating >= 1 {
				persistRating(rating, len(intent), cloudClient)
			}
			if rating >= 4 && rating <= 5 {
				break
			}
			if rating == 0 {
				break
			}
			// rating 1, 2, or 3
			if try >= maxTries {
				fmt.Fprintln(os.Stderr, "Max retries reached.")
				break
			}
			retry, err := ui.Confirm("\nWant to retry?", false)
			if err != nil || !retry {
				break
			}
			var result2 *prompt.EnhanceResult
			if showSpinner {
				done := make(chan struct{})
				go runSpinner(done, spinnerModel, rawTokenEst)
				result2, err = prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
				close(done)
			} else {
				result2, err = prompt.EnhanceWithFallback(cfg, appCfg.EnhanceURL, appCfg.EnhanceMode)
			}
			if err != nil {
				break
			}
			currentResult = result2
			if showScore {
				sc2 := prompt.ScoreEnhanceResult(cfg.Intent, result2.Prompt)
				printQualityScoreBox(sc2.Score, sc2.Hints)
			}
			fmt.Println(ui.FormatPromptForTerminal(result2.Prompt))
		}
	}

	if saveName == "" && currentResult.Prompt != "" && interactive() {
		askSaveToMemory(currentResult, appCfg, intent)
	}

	if interactive() {
		incrementCreateRunCount()
		maybeAskFeedback(cloudClient)
		maybeShowAliasTip()
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
		msg := err.Error()
		if strings.Contains(msg, "required variable") {
			return fmt.Errorf("failed to render template: %w. Run 'promptctl vars %s' for required variables", err, name)
		}
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

// openFolderInManager opens the given absolute path in the OS file manager (Finder, xdg-open, explorer).
func openFolderInManager(absPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	case "windows":
		cmd = exec.Command("explorer", absPath)
	default:
		return fmt.Errorf("open folder not supported on %s", runtime.GOOS)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open folder: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Opened %s\n", absPath)
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
	return openFolderInManager(abs)
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
		if interactive() {
			maybeOfferShellAliases()
		}
	}

	return nil
}

// maybeOfferShellAliases offers to add shell aliases (prompt, p) after init. If alias names are taken, asks for alternatives.
func maybeOfferShellAliases() {
	profile, err := shell.ProfilePath()
	if err != nil {
		return
	}
	longAlias := "prompt"
	shortAlias := "p"
	promptTaken, _ := shell.AliasExists(profile, "prompt")
	pTaken, _ := shell.AliasExists(profile, "p")
	if promptTaken || pTaken {
		skip, _ := ui.Confirm("Aliases are already set. Skip alias setup?", true)
		if skip {
			return
		}
	}
	if promptTaken {
		var custom string
		if err := ui.InputWithDefault("Alias 'prompt' is already in use. What name should we use instead? (e.g. pt)", "pt", &custom); err != nil {
			return
		}
		longAlias = strings.TrimSpace(custom)
		if longAlias == "" {
			longAlias = "pt"
		}
	}
	if pTaken {
		var custom string
		if err := ui.InputWithDefault("Alias 'p' is already in use. What name should we use instead? (e.g. pc, or leave empty to skip short alias)", "pc", &custom); err != nil {
			return
		}
		custom = strings.TrimSpace(custom)
		if custom == "" {
			shortAlias = ""
		} else {
			shortAlias = custom
		}
	}
	if shortAlias == longAlias {
		shortAlias = ""
	}
	confirmMsg := "Add shell aliases so you can run '" + longAlias + "'"
	if shortAlias != "" {
		confirmMsg += " or '" + shortAlias + "'"
	}
	confirmMsg += " instead of 'promptctl'?"
	ok, err := ui.Confirm(confirmMsg, true)
	if err != nil || !ok {
		return
	}
	if err := shell.AddAliases(profile, longAlias, shortAlias); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write to %s: %v\n", profile, err)
		return
	}
	fmt.Println(ui.Success("Aliases added to " + profile + "."))
	fmt.Fprintln(os.Stderr, ui.Hint("Use them with: "+longAlias+" create \"...\" or "+longAlias+" list"))
	if shortAlias != "" {
		fmt.Fprintln(os.Stderr, ui.Hint("Short form: "+shortAlias+" create \"...\" or "+shortAlias+" list"))
	}
	fmt.Fprintln(os.Stderr, ui.Hint("Reload your shell: source "+profile))
}

const welcomeBoxWidth = 57

func welcomeBoxLine(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > welcomeBoxWidth {
		s = s[:welcomeBoxWidth-3] + "..."
	}
	return "  │" + s + strings.Repeat(" ", welcomeBoxWidth-len(s)) + "│"
}

// runFirstTimeOnboarding runs the full first-time setup: welcome, init, default format, LLM config, shell aliases. Called once when user runs promptctl for the first time (interactive).
func runFirstTimeOnboarding() {
	fmt.Println()
	fmt.Println("  ┌" + strings.Repeat("─", welcomeBoxWidth) + "┐")
	fmt.Println(welcomeBoxLine("Welcome to promptctl"))
	fmt.Println(welcomeBoxLine("Turn raw ideas into structured prompts. Save 55-71% on LLM costs."))
	fmt.Println(welcomeBoxLine("Works with Claude, GPT-5, Groq, DeepSeek."))
	fmt.Println("  └" + strings.Repeat("─", welcomeBoxWidth) + "┘")
	fmt.Println()
	fmt.Println("  You'll go through 4 steps:")
	fmt.Println("    1) Create config and starter templates (~/.promptctl)")
	fmt.Println("    2) Choose default output format for 'promptctl create'")
	fmt.Println("    3) Set up your LLM (provider + API key)")
	fmt.Println("    4) Optionally add shell aliases (prompt, p)")
	fmt.Println()
	fmt.Println("  Let's get you set up.")
	fmt.Println()

	// Step 1: Init (create ~/.promptctl, starter templates)
	if err := config.InitGlobal(); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: init failed: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, ui.Success("  ✓ Step 1/4 — Config and starter templates ready."))
		fmt.Println()
	}

	// Step 2: Default output format for 'create' (so we don't ask every time)
	fmt.Println("  Step 2/4 — Choose default output format for 'promptctl create' (you won't be asked again).")
	var formatChoice string
	formatOptions := []string{"Markdown (recommended)", "XML", "YAML", "JSON", "Plain text"}
	if err := ui.SelectOption("  Output format", formatOptions, &formatChoice); err == nil {
		format := "markdown"
		switch formatChoice {
		case "Markdown (recommended)", "YAML", "JSON", "Plain text":
			format = "markdown"
		case "XML":
			format = "xml"
		}
		_ = config.SaveCreateFormat(format)
		fmt.Fprintln(os.Stderr, ui.Success("  ✓ Step 2/4 — Default format saved (change later with --format=...)."))
	} else {
		_ = config.SaveCreateFormat("markdown")
	}
	fmt.Println()

	// Step 3: LLM provider and API key
	fmt.Println("  Step 3/4 — Choose your LLM provider and add your API key.")
	if err := configOnboarding(); err != nil {
		fmt.Fprintf(os.Stderr, "  You can run 'promptctl config' later to set up your LLM.\n\n")
	} else {
		fmt.Fprintln(os.Stderr, ui.Success("  ✓ Step 3/4 — LLM configured."))
		fmt.Println()
	}

	// Step 4: Shell aliases (prompt, p)
	fmt.Println("  Step 4/4 — Add short commands so you don't have to type 'promptctl' every time.")
	maybeOfferShellAliases()
	fmt.Println()

	fmt.Println("  ┌─────────────────────────────────────────────────────────┐")
	fmt.Println("  │  You're all set. Try:                                    │")
	fmt.Println("  │    promptctl create \"analyze my startup idea\"            │")
	fmt.Println("  │    promptctl list                                         │")
	fmt.Println("  │  Or use 'prompt' / 'p' if you added aliases.              │")
	fmt.Println("  └─────────────────────────────────────────────────────────┘")
	fmt.Println()
}

const aliasTipShownFile = "alias_tip_shown"

// maybeShowAliasTip prints a one-time hint about prompt/p aliases if the user hasn't added them yet.
func maybeShowAliasTip() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	shownPath := filepath.Join(home, ".promptctl", aliasTipShownFile)
	if _, err := os.Stat(shownPath); err == nil {
		return
	}
	profile, err := shell.ProfilePath()
	if err != nil {
		return
	}
	hasBlock, err := shell.HasPromptctlAliasBlock(profile)
	if err != nil || hasBlock {
		return
	}
	fmt.Fprintln(os.Stderr, ui.Hint("Tip: run 'promptctl init' to add aliases (prompt, p) so you can use: prompt create \"...\" or p list"))
	dir := filepath.Join(home, ".promptctl")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(shownPath, []byte(""), 0644)
}

// latestReleaseVersion fetches the latest release tag from GitHub.
func latestReleaseVersion(timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleasesLatest, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var v struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return ""
	}
	latest := strings.TrimPrefix(strings.TrimSpace(v.TagName), "v")
	return latest
}

func autoUpdateEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PROMPTCTL_AUTOUPDATE"))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func autoUpdateOnLaunch(command string) string {
	if command == "version" || command == "-v" || command == "--version" {
		return ""
	}
	if os.Getenv("CI") != "" || !interactive() || !autoUpdateEnabled() {
		return ""
	}
	if !shouldRunAutoUpdateNow() {
		return ""
	}
	defer markAutoUpdateCheckNow()

	latest := latestReleaseVersion(3 * time.Second)
	if latest == "" || !versionLess(version, latest) {
		return ""
	}

	updater, args, err := detectUpdateCommand()
	if err != nil {
		return "A new version (v" + latest + ") is available. Run: brew upgrade promptctl"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, updater, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "Auto-update failed. Run manually: " + updater + " " + strings.Join(args, " ")
	}

	return "Auto-updated to latest available release (v" + latest + ")."
}

func detectUpdateCommand() (string, []string, error) {
	if _, err := exec.LookPath("brew"); err == nil {
		if brewHasPackage("formula", "promptctl") {
			return "brew", []string{"upgrade", "--formula", "promptctl"}, nil
		}
		if brewHasPackage("cask", "promptctl") {
			return "brew", []string{"upgrade", "--cask", "promptctl"}, nil
		}
	}
	return "", nil, fmt.Errorf("no supported package manager detected")
}

func brewHasPackage(kind, name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "brew", "list", "--"+kind, name)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func autoUpdateCheckPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".promptctl", "last_auto_update_check"), nil
}

func shouldRunAutoUpdateNow() bool {
	path, err := autoUpdateCheckPath()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return true
	}
	return time.Since(ts) >= autoUpdateInterval
}

func markAutoUpdateCheckNow() {
	path, err := autoUpdateCheckPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
}

// versionLess returns true if a is strictly less than b (e.g. "0.8.6" < "0.9.0"). Non-numeric parts are compared as strings.
func versionLess(a, b string) bool {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	for i := 0; i < len(partsA) || i < len(partsB); i++ {
		var na, nb int
		if i < len(partsA) {
			na, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			nb, _ = strconv.Atoi(partsB[i])
		}
		if na < nb {
			return true
		}
		if na > nb {
			return false
		}
	}
	return false
}

// parseVars extracts --key=value pairs and boolean flags from args
// Boolean flags (like --record) get an empty string value
func parseVars(args []string) map[string]string {
	vars := make(map[string]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			} else {
				// Boolean flag (no equals sign) - set empty string as value
				vars[parts[0]] = ""
			}
		}
	}
	return vars
}
func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
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

// stderrIsTerminal returns true if stderr is a character device (so we can show a spinner).
func stderrIsTerminal() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// interactive returns true when both stdin and stdout are terminals (no piping).
func interactive() bool {
	return stdinIsTerminal() && stdoutIsTerminal()
}

// analyzeSpinnerMessages are rotating lines shown next to the spinner (change every few seconds).
var analyzeSpinnerMessages = []string{
	"Analyzing prompt...",
	"Consulting the prompt oracle...",
	"Polishing your words...",
	"Adding structure (and savings)...",
	"Almost there...",
}

// spinnerFrames are Braille-style frames for a smooth Claude-like loader (single line, in-place update).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerLineWidth = 76

// runSpinner shows a cycling spinner glyph and rotating message on stderr until done is closed.
// modelName is shown on the right; when rawTokenEst > 0, appends " · ~N tok (raw)" (pre-optimization estimate).
func runSpinner(done <-chan struct{}, modelName string, rawTokenEst int) {
	if !stderrIsTerminal() {
		return
	}
	frameTick := time.NewTicker(80 * time.Millisecond)
	msgTick := time.NewTicker(2200 * time.Millisecond)
	defer frameTick.Stop()
	defer msgTick.Stop()
	var frameIdx, msgIdx int
	msg := analyzeSpinnerMessages[0]
	rightSuffix := modelName
	if rawTokenEst > 0 {
		rightSuffix = modelName + "  ·  ~" + formatNumSimple(rawTokenEst) + " tok (raw)"
	}
	writeLine := func(frame, text string) {
		left := "  " + frame + " " + text + "  "
		right := ""
		if rightSuffix != "" {
			right = "  " + rightSuffix
		}
		pad := spinnerLineWidth - len(left) - len(right)
		if pad < 1 {
			pad = 1
		}
		fmt.Fprintf(os.Stderr, "\r\033[K%s%s%s", left, strings.Repeat(" ", pad), right)
	}
	for {
		select {
		case <-done:
			fmt.Fprintf(os.Stderr, "\r\033[K")
			return
		case <-frameTick.C:
			frameIdx = (frameIdx + 1) % len(spinnerFrames)
			writeLine(spinnerFrames[frameIdx], msg)
		case <-msgTick.C:
			msgIdx = (msgIdx + 1) % len(analyzeSpinnerMessages)
			msg = analyzeSpinnerMessages[msgIdx]
			writeLine(spinnerFrames[frameIdx], msg)
		}
	}
}

// printQualityScoreBox writes a framed quality score and hints to stderr (bold, visible). Hints wrap to multiple lines so full text is shown.
func printQualityScoreBox(score int, hints []string) {
	const boxWidth = 56
	innerWidth := boxWidth - 4 // "│ " and " │"
	top := "┌" + strings.Repeat("─", boxWidth-2) + "┐"
	bot := "└" + strings.Repeat("─", boxWidth-2) + "┘"
	scoreLine := fmt.Sprintf("  %s  %s  ", ui.Bold("Quality score:"), ui.Success(fmt.Sprintf("%d/100", score)))
	scorePlainLen := len(stripANSI(scoreLine))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, top)
	fmt.Fprintf(os.Stderr, "│ %s%s │\n", scoreLine, strings.Repeat(" ", boxWidth-2-scorePlainLen))
	if len(hints) > 0 {
		hintText := strings.Join(hints, " · ")
		for _, line := range wrapLines(stripANSI(hintText), innerWidth) {
			hintLine := ui.Hint(line)
			linePlainLen := len(stripANSI(hintLine))
			fmt.Fprintf(os.Stderr, "│ %s%s │\n", hintLine, strings.Repeat(" ", boxWidth-2-linePlainLen))
		}
	}
	fmt.Fprintln(os.Stderr, bot)
	fmt.Fprintln(os.Stderr)
}

// wrapLines splits s into lines of at most width runes, breaking on spaces. Returns one line if s fits.
func wrapLines(s string, width int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if len(s) <= width {
		return []string{s}
	}
	var lines []string
	for s != "" {
		if len(s) <= width {
			lines = append(lines, strings.TrimSpace(s))
			break
		}
		chunk := s[:width]
		lastSpace := strings.LastIndex(chunk, " ")
		if lastSpace <= 0 {
			lastSpace = width
		}
		line := strings.TrimSpace(s[:lastSpace])
		s = strings.TrimSpace(s[lastSpace:])
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

var ratingOptions = []string{"1 - Poor", "2", "3", "4", "5 - Great", "Skip"}

func ratingFromOption(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "s" || s == "skip" {
		return 0
	}
	for i := 1; i <= 5; i++ {
		if s == strconv.Itoa(i) {
			return i
		}
	}
	// Match full labels case-insensitively (e.g. "1 - poor", "5 - great")
	for i, opt := range ratingOptions {
		if strings.EqualFold(opt, s) {
			if i == 5 {
				return 0
			}
			return i + 1
		}
	}
	// Default "5" + user typed "1" can produce "51"; take last digit so we get 1 and offer retry
	if len(s) >= 1 {
		last := s[len(s)-1]
		if last >= '1' && last <= '5' {
			return int(last - '0')
		}
	}
	return 0
}

// askUserRating prompts for a 1-5 rating. One horizontal line of options; user types 1–5 or s. Returns rating (1-5), or 0 if skipped.
func askUserRating() int {
	if !ui.Interactive() {
		return 0
	}
	fmt.Fprintf(os.Stderr, "\n  Rate this output:  %s  %s  %s  %s  %s  %s\n  ",
		ui.Hint("1"), ui.Hint("2"), ui.Hint("3"), ui.Hint("4"), ui.Hint("5"), ui.Hint("[s]kip"))
	var choice string
	if err := ui.InputWithDefault("(1-5 or s)", "5", &choice); err != nil {
		return 0
	}
	r := ratingFromOption(choice)
	if r >= 1 {
		fmt.Fprintf(os.Stderr, "Thanks — %d/5.\n", r)
	}
	return r
}

// persistRating appends a rating to ~/.promptctl/ratings.json and optionally sends it to cloud.
func persistRating(rating int, intentLen int, cloudClient cloud.Client) {
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
	if cloudClient != nil && cloudClient.Enabled() {
		go func() {
			_ = cloudClient.PostRating(rating, intentLen)
		}()
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

const (
	feedbackIntervalRuns = 10
	feedbackIntervalDays = 7
)

// incrementCreateRunCount adds 1 to the create run counter in ~/.promptctl/create_run_count.
func incrementCreateRunCount() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".promptctl")
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "create_run_count")
	var count int
	if data, err := os.ReadFile(path); err == nil {
		count, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	count++
	_ = os.WriteFile(path, []byte(strconv.Itoa(count)), 0644)
}

// shouldAskFeedback returns true if we should prompt for feedback (every N runs or every N days since last ask).
func shouldAskFeedback() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	dir := filepath.Join(home, ".promptctl")
	countPath := filepath.Join(dir, "create_run_count")
	lastPath := filepath.Join(dir, "feedback_last_asked")
	var runCount int
	if data, err := os.ReadFile(countPath); err == nil {
		runCount, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	var lastAsked string
	if data, err := os.ReadFile(lastPath); err == nil {
		lastAsked = strings.TrimSpace(string(data))
	}
	if runCount%feedbackIntervalRuns == 0 && runCount > 0 {
		return true
	}
	if lastAsked == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", lastAsked)
	if err != nil {
		return false
	}
	daysSince := int(time.Now().UTC().Sub(t).Hours() / 24)
	return daysSince >= feedbackIntervalDays
}

// recordFeedbackAsked writes today's date to ~/.promptctl/feedback_last_asked.
func recordFeedbackAsked() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".promptctl")
	_ = os.MkdirAll(dir, 0755)
	today := time.Now().UTC().Format("2006-01-02")
	_ = os.WriteFile(filepath.Join(dir, "feedback_last_asked"), []byte(today), 0644)
}

// submitFeedback sends feedback anonymously to cloud when enabled, with local fallback.
func submitFeedback(text string, cloudClient cloud.Client) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if cloudClient != nil && cloudClient.Enabled() {
		if err := cloudClient.SubmitFeedback(text); err == nil {
			recordFeedbackAsked()
			return
		}
	}
	appendFeedbackLocal(text)
}

func appendFeedbackLocal(text string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".promptctl")
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "feedback.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")
	_, _ = fmt.Fprintf(f, "%s\t%s\n", ts, text)
	recordFeedbackAsked()
}

// maybeAskFeedback prompts for anonymous freeform feedback when appropriate and submits it.
func maybeAskFeedback(cloudClient cloud.Client) {
	if !ui.Interactive() {
		return
	}
	if !shouldAskFeedback() {
		return
	}
	var text string
	_ = ui.Input("\nAny feedback for the promptctl team? (optional, Enter to skip)", &text)
	if strings.TrimSpace(text) == "" {
		return
	}
	submitFeedback(text, cloudClient)
	fmt.Fprintln(os.Stderr, "Thanks — feedback submitted anonymously.")
}

// readLineStdin reads one line from stdin (trimmed). Used for interactive prompts.
func readLineStdin() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// suggestPromptName returns a safe prompt name from intent; if name exists in entries, appends timestamp.
func suggestPromptName(intent string, entries []prompt.PromptEntry) string {
	words := strings.Fields(intent)
	const maxWords = 6
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	slug := strings.ToLower(strings.Join(words, "-"))
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return -1
	}, slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "prompt"
	}
	if len(slug) > 50 {
		slug = slug[:50]
	}
	exists := make(map[string]bool)
	for _, e := range entries {
		exists[e.Name] = true
	}
	if !exists[slug] {
		return slug
	}
	ts := time.Now().UTC().Format("200601021504")
	return slug + "-" + ts
}

// askSaveToMemory prompts interactively to save the enhanced prompt to memory (PromptsDir).
func askSaveToMemory(result *prompt.EnhanceResult, appCfg *config.Config, intent string) {
	if !ui.Interactive() {
		return
	}
	save, err := ui.Confirm("\nSave to memory?", true)
	if err != nil || !save {
		return
	}
	entries, _ := prompt.ListPromptsInDir(appCfg.PromptsDir)
	folderSet := make(map[string]bool)
	for _, e := range entries {
		if e.Folder != "" {
			folderSet[e.Folder] = true
		}
	}
	if len(folderSet) > 0 {
		folders := make([]string, 0, len(folderSet))
		for f := range folderSet {
			folders = append(folders, f)
		}
		sort.Strings(folders)
		fmt.Fprintf(os.Stderr, "Existing folders: %s\n", strings.Join(folders, ", "))
	}
	var folder, name string
	_ = ui.Input("Folder name (optional, Enter to skip)", &folder)
	folder = strings.ReplaceAll(strings.TrimSpace(folder), " ", "-")
	if folder != "" && !prompt.IsValidTemplateName(folder) {
		fmt.Fprintln(os.Stderr, "Invalid folder name (use only letters, numbers, hyphen, underscore).")
		return
	}
	suggested := suggestPromptName(intent, entries)
	_ = ui.InputWithDefault("Prompt name (Enter to use suggested, or type your own)", suggested, &name)
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")
	if name == "" {
		name = suggested
	}
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
	openPrompt := "Open folder in Finder?"
	if runtime.GOOS != "darwin" {
		openPrompt = "Open folder?"
	}
	if openIt, err := ui.Confirm("\n"+openPrompt, false); err == nil && openIt {
		_ = openFolderInManager(targetAbs)
	}
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

// copyToClipboard runs a clipboard tool with text on stdin (pbcopy, xclip, xsel, wl-copy).
func copyToClipboard(text string) error {
	tools := []struct {
		name string
		args []string
	}{
		{"pbcopy", nil}, // macOS
		{"xclip", []string{"-selection", "clipboard"}}, // Linux X11
		{"xsel", []string{"--clipboard", "--input"}},   // Linux X11 alt
		{"wl-copy", nil}, // Wayland
		{"clip", nil},    // Windows
	}
	for _, tool := range tools {
		path, err := findExecutable(tool.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, tool.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			continue
		}
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

Options: --min-score=N (0-100, refuse to send if prompt quality below N). Gemini 3.1: --thinking-level=low|high, --media-resolution=low|medium|high

Examples:
  promptctl send review --file=auth.ts --model=claude-sonnet-4.5
  promptctl send --create "analyze my startup idea about X" --model=gemini-3.1-pro --thinking-level=high`)
	}

	vars := parseVars(os.Args[2:])
	modelID := vars["model"]
	if modelID == "" {
		cfg, err := ensureLLMConfig()
		if err != nil {
			return err
		}
		if cfg != nil {
			modelID = cfg.DefaultModel
		}
		if modelID == "" {
			modelID = "claude-sonnet-4-5-20250929"
		}
	}
	delete(vars, "model")
	thinkingLevel := vars["thinking-level"]
	mediaResolution := vars["media-resolution"]
	delete(vars, "thinking-level")
	delete(vars, "media-resolution")
	minScore := 0
	if s, ok := vars["min-score"]; ok {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 100 {
			minScore = n
		}
		delete(vars, "min-score")
	}
	var completeOpts *llm.CompleteOptions
	if thinkingLevel != "" || mediaResolution != "" {
		completeOpts = &llm.CompleteOptions{ThinkingLevel: thinkingLevel, MediaResolution: mediaResolution}
	}

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

	// Prompt quality score (for desired output: higher score = better prompt structure)
	pq := prompt.ScorePromptQuality(renderedPrompt)
	if minScore > 0 && pq.Score < minScore {
		fmt.Fprintf(os.Stderr, "\n  Prompt quality: %d/100 (below --min-score=%d)\n", pq.Score, minScore)
		if len(pq.Rules) > 0 {
			fmt.Fprintf(os.Stderr, "  Issues: %s\n", strings.Join(pq.Rules, ", "))
		}
		return fmt.Errorf("prompt quality %d below minimum %d; improve prompt or run without --min-score", pq.Score, minScore)
	}

	// Show cost estimate before sending
	est, err := llm.EstimateCost(renderedPrompt, modelID, promptType)
	if err != nil {
		return fmt.Errorf("cost estimation failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n  Model:  %s\n", est.ModelName)
	fmt.Fprintf(os.Stderr, "  Prompt quality: %d/100\n", pq.Score)
	fmt.Fprintf(os.Stderr, "  Tokens: ~%s in / ~%s out (optimized)\n", formatNumSimple(est.InputTokens), formatNumSimple(est.EstOutputTokens))
	fmt.Fprintf(os.Stderr, "  Without promptctl: ~$%.4f (typical rework)\n", est.WastedWithout)
	fmt.Fprintf(os.Stderr, "  Est. cost: $%.4f (saves ~$%.4f)\n\n", est.TotalEstCost, est.Savings)

	// Execute the LLM call
	result, err := llm.CompleteWithOptions(renderedPrompt, modelID, completeOpts)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Print the response
	fmt.Println(result.Content)

	// Print cost summary to stderr
	savedVsUnstructured := result.ActualCost * 2 // fallback
	if m, err := llm.FindModel(modelID); err == nil {
		mult := llm.UnstructuredMultiplier(m.InputPerMTok)
		savedVsUnstructured = result.ActualCost * (mult - 1)
	}
	withoutCost := result.ActualCost + savedVsUnstructured
	fmt.Fprintf(os.Stderr, "\n  ─── Cost Report ───\n")
	fmt.Fprintf(os.Stderr, "  Model:    %s\n", result.Model)
	fmt.Fprintf(os.Stderr, "  Tokens:   %s in / %s out (used)\n",
		formatNumSimple(result.InputTokens), formatNumSimple(result.OutputTokens))
	fmt.Fprintf(os.Stderr, "  Cost:     $%.4f\n", result.ActualCost)
	fmt.Fprintf(os.Stderr, "  Without promptctl: ~$%.4f (typical)\n", withoutCost)
	fmt.Fprintf(os.Stderr, "  Latency:  %.1fs\n", float64(result.LatencyMs)/1000)
	fmt.Fprintf(os.Stderr, "  Saved:    ~$%.4f vs unstructured\n\n", savedVsUnstructured)

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
	var modelOverride string
	for _, a := range os.Args[2:] {
		if strings.HasPrefix(a, "--calls-per-day=") {
			var n int
			if _, err := fmt.Sscanf(strings.TrimPrefix(a, "--calls-per-day="), "%d", &n); err == nil && n > 0 && n <= 1000 {
				callsPerDay = n
			}
		}
		if strings.HasPrefix(a, "--model=") {
			modelOverride = strings.TrimPrefix(a, "--model=")
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
	if modelOverride != "" {
		modelID = modelOverride
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
	if modelOverride != "" {
		fmt.Printf("  Model: %s\n", modelName)
	} else {
		fmt.Printf("  Model: %s (default)\n", modelName)
	}
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
	type indexedModel struct {
		model    llm.Model
		provider string
		key      string
	}
	var allModels []indexedModel
	for _, key := range llm.ProviderKeys() {
		provider := llm.Providers[key]
		for _, model := range provider.Models {
			allModels = append(allModels, indexedModel{model: model, provider: provider.Name, key: key})
		}
	}

	options := make([]string, len(allModels))
	for i, m := range allModels {
		marker := "  "
		if m.model.ID == cfg.DefaultModel {
			marker = "▸ "
		}
		keyStatus := "  ✗ no key"
		if getAPIKeyStatus(m.model.Provider, cfg) {
			keyStatus = "  ✓"
		}
		options[i] = fmt.Sprintf("%s%-12s %-22s $%.2f/MTok in  $%.2f/MTok out%s",
			marker, m.provider, m.model.Name, m.model.InputPerMTok, m.model.OutputPerMTok, keyStatus)
	}

	var choice string
	if ui.Interactive() {
		if err := ui.SelectOption("  Select your default model", options, &choice); err != nil {
			return err
		}
	} else {
		fmt.Print("  Select your default model (run in a terminal for interactive selection):\n")
		for _, o := range options {
			fmt.Println("   ", o)
		}
		fmt.Print("\n  Enter number (or 'q' to cancel): ")
		var input string
		fmt.Scanln(&input)
		if input == "q" || input == "" {
			return nil
		}
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(allModels) {
			return fmt.Errorf("invalid selection")
		}
		choice = options[idx-1]
	}

	idx := indexOf(options, choice)
	if idx < 0 {
		return fmt.Errorf("invalid selection")
	}
	selected := allModels[idx]
	cfg.DefaultModel = selected.model.ID
	cfg.DefaultProvider = selected.model.Provider

	if !getAPIKeyStatus(selected.model.Provider, cfg) {
		fmt.Printf("\n  ⚠  No API key for %s.\n", selected.provider)
		fmt.Printf("  Run: promptctl config  (to set up your API key)\n\n")
	}

	if err := llm.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n  %s\n\n", ui.Success("✓ Default model set to: "+selected.model.Name+" ("+selected.model.ID+")"))
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

// ensureLLMConfig returns LLM config, running onboarding when missing and TTY. When not TTY and config missing, returns error.
func ensureLLMConfig() (*llm.Config, error) {
	cfg, err := llm.LoadConfig()
	if err != nil {
		cfg = nil
	}
	if cfg != nil && cfg.DefaultModel != "" && getAPIKeyStatus(cfg.DefaultProvider, cfg) {
		return cfg, nil
	}
	if !ui.Interactive() {
		return nil, fmt.Errorf("no LLM config. Run `promptctl config` in a terminal to set up")
	}
	if onboarding.OnboardingSkipped() {
		fmt.Fprintln(os.Stderr, ui.Hint(onboarding.ReminderMessage()))
	}
	if err := configOnboarding(); err != nil {
		return nil, err
	}
	return llm.LoadConfig()
}

// configOnboarding is the full interactive setup wizard (Survey when TTY).
func configOnboarding() error {
	if !ui.Interactive() {
		return fmt.Errorf("run promptctl config in a terminal to set up")
	}
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
	providerKeys := llm.ProviderKeys()
	providerOptions := make([]string, len(providerKeys))
	for i, key := range providerKeys {
		p := llm.Providers[key]
		providerOptions[i] = p.Name + "  " + getProviderPriceRange(p)
	}
	var providerChoice string
	if err := ui.SelectOption("  Step 1/4 - Choose your LLM provider", providerOptions, &providerChoice); err != nil {
		_ = onboarding.MarkOnboardingSkipped()
		return err
	}
	providerIdx := indexOf(providerOptions, providerChoice)
	if providerIdx < 0 {
		_ = onboarding.MarkOnboardingSkipped()
		return fmt.Errorf("invalid selection")
	}
	selectedProviderKey := providerKeys[providerIdx]
	selectedProvider := llm.Providers[selectedProviderKey]
	fmt.Fprintf(os.Stderr, "\n  %s\n\n", ui.Success("✓ Provider: "+selectedProvider.Name))

	// ── Step 2: Select model ─────────────────────────────────────
	modelOptions := make([]string, len(selectedProvider.Models))
	for i, m := range selectedProvider.Models {
		modelOptions[i] = fmt.Sprintf("%s  $%.2f/MTok in  $%.2f/MTok out  %sk context",
			m.Name, m.InputPerMTok, m.OutputPerMTok, formatNumSimple(m.ContextWindow/1000))
	}
	bestValue := findBestValue(selectedProvider)
	fmt.Printf("  Recommended: %s (best price/quality ratio)\n\n", bestValue.Name)
	var modelChoice string
	if err := ui.SelectOption("  Step 2/4 - Choose your default model", modelOptions, &modelChoice); err != nil {
		_ = onboarding.MarkOnboardingSkipped()
		return err
	}
	modelIdx := indexOf(modelOptions, modelChoice)
	if modelIdx < 0 {
		_ = onboarding.MarkOnboardingSkipped()
		return fmt.Errorf("invalid selection")
	}
	selectedModel := selectedProvider.Models[modelIdx]
	fmt.Fprintf(os.Stderr, "\n  %s\n\n", ui.Success("✓ Model: "+selectedModel.Name))

	// ── Step 3: API key (skip for Atlas / promptctl — no key needed) ──
	if selectedProviderKey == "promptctl" {
		fmt.Println("\n  ✓ Atlas (hosted) — no API key needed")
	} else {
		var keyInput string
		existingKey := llm.GetAPIKey(selectedProviderKey, selectedProvider.EnvKey)
		if existingKey != "" {
			keep, err := ui.Confirm("  You already have a key configured: "+maskKey(existingKey)+"\n  Keep existing key?", true)
			if err != nil {
				_ = onboarding.MarkOnboardingSkipped()
				return err
			}
			if keep {
				fmt.Println("\n  ✓ Keeping existing API key")
				goto saveConfig
			}
		}
		fmt.Printf("  To get your API key, open: %s\n\n", selectedProvider.KeyURL)
		fmt.Fprint(os.Stderr, "  Press Enter to open the API key page in your browser... ")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		openBrowser(selectedProvider.KeyURL)
		if err := ui.Password("  Paste your API key and press Enter to save", &keyInput); err != nil {
			_ = onboarding.MarkOnboardingSkipped()
			return err
		}
		if strings.TrimSpace(keyInput) == "" {
			_ = onboarding.MarkOnboardingSkipped()
			return fmt.Errorf("no API key provided. Run 'promptctl config' to try again")
		}
		if err := llm.SetAPIKey(selectedProviderKey, strings.TrimSpace(keyInput)); err != nil {
			_ = onboarding.MarkOnboardingSkipped()
			return fmt.Errorf("saving API key: %w", err)
		}
		if runtime.GOOS == "darwin" {
			fmt.Printf("\n  ✓ API key saved to Keychain\n")
		} else {
			fmt.Printf("\n  ✓ API key stored securely (~/.promptctl/llm.json)\n")
		}
	}

saveConfig:
	cfg, err = llm.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	cfg.DefaultProvider = selectedProviderKey
	cfg.DefaultModel = selectedModel.ID

	if err := llm.SaveConfig(cfg); err != nil {
		_ = onboarding.MarkOnboardingSkipped()
		return fmt.Errorf("failed to save config: %w", err)
	}
	_ = onboarding.ClearOnboardingSkipped()

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
	name  string
	args  []string
	Stdin io.Reader
}

func (c *execCmd) Run() error {
	cmd := fmt.Sprintf("%s %s", c.name, strings.Join(c.args, " "))
	_ = cmd
	return nil // best effort, don't fail if stty unavailable
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "linux":
		c = exec.Command("xdg-open", url)
	case "windows":
		c = exec.Command("cmd", "/c", "start", "", url)
	default:
		return
	}
	if c != nil {
		_ = c.Start() // best effort; don't block on browser
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
