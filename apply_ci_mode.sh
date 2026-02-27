#!/usr/bin/env bash
set -e

echo "===== Adding CI Mode ====="

FILE="cmd/experiment.go"

cp $FILE ${FILE}.bak

echo "Injecting CI flag parsing..."

# Insert ci flag parsing after strict-json parsing
sed -i '' '/strictJSON := false/a\
\
\tciMode := false\
\tif _, ok := vars["ci"]; ok {\
\t\tciMode = true\
\t\tdelete(vars, "ci")\
\t}' $FILE

echo "Injecting CI output handling..."

# Replace final summary block with CI-aware logic
sed -i '' '/fmt.Println("Summary:")/,$d' $FILE

cat >> $FILE << 'EOF'

	// CI Mode Output
	if ciMode {

		pass := true
		if len(results) == 0 {
			pass = false
		} else {
			if minScore > 0 && results[0].Score < minScore {
				pass = false
			}
		}

		type ciSummary struct {
			Model     string  `json:"model"`
			Score     int     `json:"score"`
			Cost      float64 `json:"cost"`
			LatencyMs int64   `json:"latency_ms"`
			Pass      bool    `json:"pass"`
		}

		if len(results) > 0 {
			best := results[0]
			summary := ciSummary{
				Model:     best.Model,
				Score:     best.Score,
				Cost:      best.Cost,
				LatencyMs: best.LatencyMs,
				Pass:      pass,
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

	fmt.Println("Summary:")
	fmt.Println("Model                      Score  Cost      Latency   Fail")
	fmt.Println("----------------------------------------------------------------")

	for _, r := range results {
		fmt.Printf("%-25s %-6d $%-8.4f  %4.1fs  %d\n",
			r.Model,
			r.Score,
			r.Cost,
			float64(r.LatencyMs)/1000,
			r.Failures,
		)
	}

	fmt.Println()
	return nil
}
EOF

echo "Formatting..."
go fmt ./...

echo "Building..."
go build -o promptctl

echo "===== CI Mode Added ====="
