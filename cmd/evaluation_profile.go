package cmd

type EvaluationProfile struct {

	// Text structure fallback
	RequiredSections []string `yaml:"required_sections"`
	RequireLineRefs  bool     `yaml:"require_line_refs"`
	MinLength        int      `yaml:"min_length"`

	// JSON enforcement
	RequireJSON      bool     `yaml:"require_json"`
	RequiredJSONKeys []string `yaml:"required_json_keys"`

	MaxCost float64 `yaml:"max_cost"`

	WeightStructure   int  `yaml:"weight_structure"`
	WeightSpecificity int  `yaml:"weight_specificity"`
	WeightDepth       int  `yaml:"weight_depth"`
	WeightEfficiency  int  `yaml:"weight_efficiency"`
	StrictJSON        bool `yaml:"strict_json"`
}

func DefaultProfile() EvaluationProfile {
	return EvaluationProfile{
		RequireJSON: true,
		RequiredJSONKeys: []string{
			"summary",
			"critical_issues",
			"performance_issues",
			"maintainability_issues",
			"line_suggestions",
		},
		MinLength: 500, // 800 is too aggressive for CI
		MaxCost:   0.02,

		WeightStructure:   40,
		WeightSpecificity: 20,
		WeightDepth:       20,
		WeightEfficiency:  20,
		StrictJSON:        true,
	}
}
