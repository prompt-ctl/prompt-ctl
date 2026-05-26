package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/prompt-ctl/promptctl/llm"
)

func runTest() error {
	return runTestWithClient(llm.RealClient{})
}

func printTestHelp() {
	fmt.Println(`USAGE:
  promptctl test <template> [--var=value ...] --models=m1,m2 [--baseline=version] [--record] [--ci]

DESCRIPTION:
  'test' is the product-facing command for prompt testing.
  It runs the same engine as 'experiment' to support:
    - model comparison (--models)
    - regression checks (--baseline)
    - benchmark recording (--record)
    - CI pass/fail behavior (--ci, --min-score)

OPTIONS:
  --models       Comma-separated list of models (required)
  --model        Single model shorthand (converted to --models)
  --repeat       Number of runs per model (default: 1)
  --min-score    Fail if best score is below threshold (0-100)
  --baseline     Compare against specific template version (single model only)
  --record       Record current score to benchmarks in meta.json (single model only)
  --ci           Print machine-readable output and use CI-friendly exit behavior

EXAMPLES:
  promptctl test review --file=main.go --models=claude-sonnet-4-5
  promptctl test review --file=main.go --model=claude-sonnet-4-5 --baseline=v1
  promptctl test review --file=main.go --model=claude-sonnet-4-5 --record --ci`)
}

func runTestWithClient(client llm.Client) error {
	if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		printTestHelp()
		return nil
	}

	if len(os.Args) < 3 {
		return fmt.Errorf(`usage:
  promptctl test <template> [--var=value ...] --models=m1,m2`)
	}

	translated := make([]string, 0, len(os.Args))
	translated = append(translated, os.Args[0], "experiment", os.Args[2])
	for _, arg := range os.Args[3:] {
		if strings.HasPrefix(arg, "--model=") {
			translated = append(translated, "--models="+strings.TrimPrefix(arg, "--model="))
			continue
		}
		translated = append(translated, arg)
	}

	oldArgs := os.Args
	os.Args = translated
	defer func() { os.Args = oldArgs }()

	return runExperimentWithClient(client)
}
