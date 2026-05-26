package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/prompt-ctl/promptctl/config"
	"github.com/prompt-ctl/promptctl/llm"
	"github.com/prompt-ctl/promptctl/prompt"
)

func runExperiment() error {
	return runExperimentWithClient(llm.RealClient{})
}

func printExperimentHelp() {
	fmt.Println(`USAGE:
  promptctl experiment <template> [--var=value ...] --models=m1,m2 [--baseline=version] [--record]

OPTIONS:
  --models       Comma-separated list of models
  --repeat       Number of runs per model (default: 1)
  --min-score    Fail if best score is below threshold (0-100)
  --baseline     Compare against specific template version (requires single model)
  --record       Record current score to benchmarks in meta.json (requires single model)

EXAMPLES:
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5 --repeat=3
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5 --min-score=70
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5 --baseline=v1
  promptctl experiment review --file=main.go --models=claude-sonnet-4-5 --record

BASELINE COMPARISON:
  The --baseline flag allows comparing the current template against a previous version.
  When used, output shows:
    Baseline (<version>): <baseline_score>
    Current:             <current_score>
    Diff:                <score_difference>
  Note: --baseline requires exactly one model to be specified.

BENCHMARK RECORDING:
  The --record flag saves the score to meta.json for future baseline comparisons.
  Constraints:
    - Requires exactly one model
    - In CI mode: only records on pass (no regression)
    - In normal mode: always records on success
  Output: "Benchmark recorded for <model>: <score>"`)
}

func runExperimentWithClient(client llm.Client) error {
	if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		printExperimentHelp()
		return nil
	}

	if len(os.Args) < 3 {
		return fmt.Errorf(`usage:
  promptctl experiment <template> [--var=value ...] --models=m1,m2`)
	}

	name := os.Args[2]
	vars := parseVars(os.Args[3:])

	strictJSON := false
	if _, ok := vars["strict-json"]; ok {
		strictJSON = true
		delete(vars, "strict-json")
	}

	ciMode := false
	if _, ok := vars["ci"]; ok {
		ciMode = true
		delete(vars, "ci")
	}

	recordBenchmark := false
	if _, ok := vars["record"]; ok {
		recordBenchmark = true
		delete(vars, "record")
	}

	repeat := 1
	minScore := 0

	if r, ok := vars["repeat"]; ok {
		_, _ = fmt.Sscanf(r, "%d", &repeat)
		delete(vars, "repeat")
	}
	if m, ok := vars["min-score"]; ok {
		_, _ = fmt.Sscanf(m, "%d", &minScore)
		delete(vars, "min-score")
	}

	if repeat < 1 {
		repeat = 1
	}
	if repeat > 20 {
		repeat = 20
	}
	if minScore < 0 {
		minScore = 0
	}
	if minScore > 100 {
		minScore = 100
	}

	modelsArg := vars["models"]
	if modelsArg == "" {
		return fmt.Errorf("missing --models")
	}
	delete(vars, "models")

	modelIDs := strings.Split(modelsArg, ",")

	// Parse --baseline flag
	baselineVersion := ""
	if b, ok := vars["baseline"]; ok {
		baselineVersion = b
		delete(vars, "baseline")
	}

	// Validate baseline + model constraint: baseline requires exactly 1 model
	if baselineVersion != "" {
		if len(modelIDs) > 1 {
			return fmt.Errorf("--baseline requires exactly one model")
		}
	}

	// Validate --record requires exactly 1 model
	if recordBenchmark {
		if len(modelIDs) > 1 {
			return fmt.Errorf("--record requires exactly one model")
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	tmpl, err := prompt.LoadTemplate(name, cfg)
	if err != nil {
		return fmt.Errorf("template '%s' not found", name)
	}

	if err := enrichFileVars(vars); err != nil {
		return err
	}

	renderedPrompt, err := tmpl.Render(vars)
	if err != nil {
		return err
	}

	// Load baseline template if specified
	var baselineRenderedPrompt string
	if baselineVersion != "" {
		baselineTemplate, err := loadSpecificTemplateVersion(name, baselineVersion, cfg)
		if err != nil {
			return err
		}

		// Render baseline using same enriched vars
		baselineRenderedPrompt, err = baselineTemplate.Render(vars)
		if err != nil {
			return err
		}
	}

	fmt.Printf("\nPrompt quality: %d/100\n\n",
		prompt.ScorePromptQuality(renderedPrompt).Score)

	type result struct {
		Model         string
		AvgScore      int
		AvgCost       float64
		AvgLatency    int64
		Failures      int
		BaselineScore int // New field for baseline comparison
		Diff          int // New field for score diff
	}

	var results []result

	for _, modelID := range modelIDs {

		modelID = strings.TrimSpace(modelID)

		fmt.Printf("Running %s (%dx)...\n", modelID, repeat)

		// Set up evaluation profile
		profile := DefaultProfile()
		profile.StrictJSON = strictJSON

		// Run current version
		currentScore, currentCost, currentLatency, currentFailures, err := runAndAggregate(
			client,
			renderedPrompt,
			modelID,
			repeat,
			profile,
		)
		if err != nil {
			return err
		}

		// Run baseline if specified
		baselineScore := 0
		if baselineVersion != "" {
			baselineScore, _, _, _, err = runAndAggregate(
				client,
				baselineRenderedPrompt,
				modelID,
				repeat,
				profile,
			)
			if err != nil {
				return fmt.Errorf("baseline experiment failed: %w", err)
			}
		}

		// Calculate diff
		diff := currentScore - baselineScore

		results = append(results, result{
			Model:         modelID,
			AvgScore:      currentScore,
			AvgCost:       currentCost,
			AvgLatency:    currentLatency,
			Failures:      currentFailures,
			BaselineScore: baselineScore,
			Diff:          diff,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgScore > results[j].AvgScore
	})

	// Print baseline comparison if specified (non-CI mode only)
	if baselineVersion != "" && len(results) > 0 && !ciMode {
		best := results[0]
		fmt.Printf("\nBaseline (%s): %d\n", baselineVersion, best.BaselineScore)
		fmt.Printf("Current:       %d\n", best.AvgScore)
		diff := best.AvgScore - best.BaselineScore
		sign := "+"
		if diff < 0 {
			sign = ""
		}
		fmt.Printf("Diff:          %s%d\n", sign, diff)
	}

	// ---------- CI MODE ----------
	if ciMode {

		if len(results) > 0 {
			best := results[0]

			// Calculate pass status: diff >= 0 AND minScore check
			pass := true
			if minScore > 0 && best.AvgScore < minScore {
				pass = false
			}

			// Baseline regression check: if baseline set, must not regress
			if baselineVersion != "" && best.AvgScore < best.BaselineScore {
				pass = false
			}

			// Record benchmark if requested and pass == true
			// In CI mode: only record if pass == true
			if recordBenchmark && pass {
				modelID := modelIDs[0]
				score := best.AvgScore
				if err := updateBenchmark(name, modelID, score, cfg); err != nil {
					return fmt.Errorf("failed to record benchmark: %w", err)
				}
			}

			type ciSummary struct {
				Model         string  `json:"model"`
				Score         int     `json:"score"`
				BaselineScore int     `json:"baseline_score,omitempty"`
				Diff          int     `json:"diff,omitempty"`
				Cost          float64 `json:"cost"`
				LatencyMs     int64   `json:"latency_ms"`
				Pass          bool    `json:"pass"`
			}

			summary := ciSummary{
				Model:         best.Model,
				Score:         best.AvgScore,
				BaselineScore: best.BaselineScore,
				Diff:          best.Diff,
				Cost:          best.AvgCost,
				LatencyMs:     best.AvgLatency,
				Pass:          pass,
			}

			b, _ := json.Marshal(summary)
			fmt.Println(string(b))

			if !pass {
				os.Exit(1)
			}
		} else {
			fmt.Println(`{"pass":false}`)
			os.Exit(1)
		}

		return nil
	}

	// ---------- NORMAL MODE ----------
	fmt.Println("Summary:")
	fmt.Println("Model                      Score  Cost      Latency   Fail")
	fmt.Println("----------------------------------------------------------------")

	for _, r := range results {
		fmt.Printf("%-25s %-6d $%-8.4f  %4.1fs  %d\n",
			r.Model,
			r.AvgScore,
			r.AvgCost,
			float64(r.AvgLatency)/1000,
			r.Failures,
		)
	}

	// Record benchmark if requested and execution was successful
	// In non-CI mode: always record on success
	if recordBenchmark && len(results) > 0 {
		modelID := modelIDs[0]
		score := results[0].AvgScore
		if err := updateBenchmark(name, modelID, score, cfg); err != nil {
			return fmt.Errorf("failed to record benchmark: %w", err)
		}

		fmt.Printf("Benchmark recorded for %s: %d\n", modelID, score)
	}

	fmt.Println()
	return nil
}
