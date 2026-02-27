package cmd

func generateVariants(base string, n int) []string {

	if n <= 1 {
		return []string{base}
	}

	var variants []string

	variants = append(variants, base)

	variants = append(variants,
		base+"\n\nBe concise. Use clear structure. Avoid repetition.",
	)

	variants = append(variants,
		base+"\n\nReturn strictly valid JSON only. No explanations.",
	)

	variants = append(variants,
		"Think step by step.\n\n"+base,
	)

	variants = append(variants,
		base+"\n\nBe critical. Explicitly identify weaknesses and risks.",
	)

	variants = append(variants,
		base+"\n\nAssume hostile attacker mindset. Identify exploit scenarios explicitly.",
	)

	variants = append(variants,
		base+"\n\nQuantify risk levels (High/Medium/Low) for each issue.",
	)

	variants = append(variants,
		base+"\n\nInclude concrete code diffs when suggesting fixes.",
	)

	if len(variants) > n {
		return variants[:n]
	}

	return variants
}
