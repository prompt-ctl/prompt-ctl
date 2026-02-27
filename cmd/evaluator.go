package cmd

import (
	"encoding/json"
	"strings"
)

func Evaluate(content string, cost float64, latencyMs int64, profile EvaluationProfile) (int, bool) {

	score := 0
	jsonInvalid := false

	// ---------- JSON MODE ----------
	if profile.RequireJSON {

		parsed, err := extractJSON(content)
		if err == nil {

			structureScore := 0
			for _, key := range profile.RequiredJSONKeys {
				if _, ok := parsed[key]; ok {
					structureScore++
				}
			}

			if len(profile.RequiredJSONKeys) > 0 {
				score += (structureScore * profile.WeightStructure) /
					len(profile.RequiredJSONKeys)
			}

			if v, ok := parsed["line_suggestions"]; ok {
				if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
					score += profile.WeightSpecificity
				}
			}

		} else {
			jsonInvalid = true
		}
	}

	// ---------- DEPTH ----------
	if len(content) >= profile.MinLength {
		score += profile.WeightDepth
	}

	// ---------- EFFICIENCY ----------
	if cost <= profile.MaxCost {
		score += profile.WeightEfficiency / 2
	}

	if latencyMs < 4000 {
		score += profile.WeightEfficiency / 2
	}

	if score > 100 {
		score = 100
	}

	return score, jsonInvalid
}

func extractJSON(content string) (map[string]interface{}, error) {

	// 1. Try fenced block first
	startFence := strings.Index(content, "```")
	if startFence != -1 {
		endFence := strings.LastIndex(content, "```")
		if endFence > startFence {
			block := content[startFence+3 : endFence]
			block = strings.TrimSpace(block)

			// Remove optional language tag (e.g., "json")
			if strings.HasPrefix(strings.ToLower(block), "json") {
				block = block[4:]
				block = strings.TrimSpace(block)
			}

			content = block
		}
	}

	// 2. Fallback: extract substring between first '{' and last '}'
	firstBrace := strings.Index(content, "{")
	lastBrace := strings.LastIndex(content, "}")
	if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
		content = content[firstBrace : lastBrace+1]
	}

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(content), &parsed)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}
