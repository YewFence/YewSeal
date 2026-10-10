package config

// ExcludeRule identifies a loaded exclusion and its one-based position in its config.
type ExcludeRule struct {
	Pattern    string
	ConfigPath string
	Index      int
}

// GetExcludeRules returns the original rules in config loading and declaration order.
func (c *Config) GetExcludeRules() []ExcludeRule {
	rules := make([]ExcludeRule, 0)
	if len(c.LoadedFiles) == 0 {
		for i, pattern := range c.Exclude {
			rules = append(rules, ExcludeRule{Pattern: pattern, Index: i + 1})
		}
		return rules
	}
	for _, file := range c.LoadedFiles {
		for i, pattern := range file.Exclude {
			rules = append(rules, ExcludeRule{Pattern: pattern, ConfigPath: file.Path, Index: i + 1})
		}
	}
	return rules
}

func (c *Config) groupExclude(group GroupConfig) []string {
	if group.ConfigPath == "" {
		return c.Exclude
	}
	for _, file := range c.LoadedFiles {
		if file.Path == group.ConfigPath {
			return file.Exclude
		}
	}
	return nil
}
