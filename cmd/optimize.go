package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/prompt-ctl/promptctl/config"
	"github.com/prompt-ctl/promptctl/llm"
	"github.com/prompt-ctl/promptctl/prompt"
)

func runOptimize() error {
	return runOptimizeWithClient(llm.RealClient{})
}

func printOptimizeHelp() {
	fmt.Println(`USAGE:
  promptctl experiment optimize <template> [--var=value ...] --models=m1

OPTIONS:
  --variants     Number of prompt variants (default: 5)
  --repeat       Runs per variant (default: 1)
  --strict-json  Strict JSON mode (default: false)

EXAMPLE:
  promptctl experiment optimize review --file=main.go --models=claude-sonnet-4-5 --variants=5 --repeat=2 --strict-json=true`)
}

func saveNewVersion(templateName string, newBody string) error {

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	versionDir := filepath.Join(cfg.LocalTemplateDir, templateName)
	metaPath := filepath.Join(versionDir, "meta.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("template not versioned")
	}

	var meta TemplateMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	// Compute next version
	lastVersion := meta.Versions[len(meta.Versions)-1]
	var nextNum int
	fmt.Sscanf(lastVersion, "v%d", &nextNum)
	nextNum++
	nextVersion := fmt.Sprintf("v%d", nextNum)

	newPath := filepath.Join(versionDir, nextVersion+".yaml")

	// Write new template file
	if err := os.WriteFile(newPath, []byte(newBody), 0644); err != nil {
		return err
	}

	meta.Current = nextVersion
	meta.Versions = append(meta.Versions, nextVersion)

	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return err
	}

	fmt.Println("Saved new version:", nextVersion)

	return nil
}

func runOptimizeWithClient(client llm.Client) error {

	if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		printOptimizeHelp()
		return nil
	}

	if len(os.Args) < 4 {
		return fmt.Errorf(`usage:
  promptctl experiment optimize <template> [--var=value ...] --models=m1`)
	}

	templateName := os.Args[3]
	vars := parseVars(os.Args[4:])

	saveFlag := false
	if _, ok := vars["save"]; ok {
		saveFlag = true
		delete(vars, "save")
	}

	strictJSON := false
	if _, ok := vars["strict-json"]; ok {
		strictJSON = true
		delete(vars, "strict-json")
	}

	variantCount := 5
	repeat := 1

	if v, ok := vars["variants"]; ok {
		fmt.Sscanf(v, "%d", &variantCount)
		delete(vars, "variants")
	}

	if r, ok := vars["repeat"]; ok {
		fmt.Sscanf(r, "%d", &repeat)
		delete(vars, "repeat")
	}

	if variantCount < 1 {
		variantCount = 1
	}
	if variantCount > 10 {
		variantCount = 10
	}
	if repeat < 1 {
		repeat = 1
	}
	if repeat > 10 {
		repeat = 10
	}

	modelsArg := vars["models"]
	if modelsArg == "" {
		return fmt.Errorf("missing --models (optimize supports one model)")
	}
	delete(vars, "models")

	modelID := strings.TrimSpace(modelsArg)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	tmpl, err := prompt.LoadTemplate(templateName, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	if err := enrichFileVars(vars); err != nil {
		return err
	}

	renderedPrompt, err := tmpl.Render(vars)
	if err != nil {
		return err
	}

	fmt.Printf("\nGenerating %d variants...\n", variantCount)

	variants := generateVariants(renderedPrompt, variantCount)

	type variantResult struct {
		Prompt string
		Score  int
	}

	var results []variantResult

	for i, v := range variants {

		fmt.Printf("Evaluating variant %d/%d...\n", i+1, len(variants))

		totalScore := 0
		totalCost := 0.0
		failures := 0
		var scores []int

		for r := 0; r < repeat; r++ {

			exec, err := runSingleExecution(client, v, modelID)
			if err != nil || exec == nil {
				fmt.Println("Execution error:", err)

				failures++
				continue
			}

			profile := DefaultProfile()
			profile.StrictJSON = strictJSON

			score, jsonInvalid := Evaluate(exec.Content, exec.Cost, exec.LatencyMs, profile)

			if jsonInvalid && profile.StrictJSON {
				failures++
				continue
			}

			totalScore += score
			totalCost += exec.Cost
			scores = append(scores, score)
		}

		success := repeat - failures
		if success <= 0 {
			success = 1
		}

		avgScore := totalScore / success
		avgCost := totalCost / float64(success)

		// Cost penalty
		costPenalty := 0
		if avgCost > 0.02 {
			costPenalty = 5
		}

		// Variance penalty
		variancePenalty := 0
		if len(scores) > 1 {
			min := scores[0]
			max := scores[0]
			for _, s := range scores {
				if s < min {
					min = s
				}
				if s > max {
					max = s
				}
			}
			if max-min > 30 {
				variancePenalty = 5
			}
		}

		effectiveScore := avgScore - costPenalty - variancePenalty
		if effectiveScore < 0 {
			effectiveScore = 0
		}

		results = append(results, variantResult{
			Prompt: v,
			Score:  effectiveScore,
		})

		fmt.Printf("  Avg Score: %d (cost penalty: %d, variance penalty: %d)\n",
			effectiveScore, costPenalty, variancePenalty)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	fmt.Println("\nBest Variant Score:", results[0].Score)
	fmt.Println("Best Prompt:")
	fmt.Println(results[0].Prompt)

	if saveFlag {
		err := saveNewVersion(templateName, results[0].Prompt)
		if err != nil {
			return err
		}
	}

	fmt.Println()
	return nil
}
