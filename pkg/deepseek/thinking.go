package deepseek

// ThinkingConfig controls DeepSeek Reasoner thinking behavior.
type ThinkingConfig struct {
	Budget  int    `json:"budget"`  // token budget for thinking (0 = auto)
	Mode    string `json:"mode"`    // "auto", "enabled", "disabled"
	Visible bool   `json:"visible"` // show thinking in UI
}

func DefaultThinkingConfig() ThinkingConfig {
	return ThinkingConfig{
		Budget:  0,
		Mode:    "auto",
		Visible: true,
	}
}

// ShouldEnableThinking returns true if the model supports thinking and it's enabled.
func ShouldEnableThinking(model string, config ThinkingConfig) bool {
	switch model {
	case "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro":
	default:
		return false
	}
	return config.Mode == "auto" || config.Mode == "enabled"
}

// ThinkingTokenBudget returns the token budget to pass to the API for thinking.
func ThinkingTokenBudget(config ThinkingConfig) *int {
	if config.Budget <= 0 {
		return nil // use API default
	}
	return &config.Budget
}
