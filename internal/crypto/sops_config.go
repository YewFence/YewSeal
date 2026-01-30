package crypto

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// SopsConfig represents the structure of .sops.yaml
type SopsConfig struct {
	CreationRules []CreationRule `yaml:"creation_rules"`
}

// CreationRule represents a single rule in .sops.yaml
type CreationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

// UpdateSopsYaml creates or updates .sops.yaml with proper YAML handling
func UpdateSopsYaml(outputFile, publicKey string, force bool) error {
	const sopsYamlPath = ".sops.yaml"

	// Generate regex pattern that precisely matches the output file
	escapedOutput := regexp.QuoteMeta(outputFile)
	pathRegex := "^" + escapedOutput + "$"

	var config SopsConfig

	// Try to read existing file
	if existingData, err := os.ReadFile(sopsYamlPath); err == nil {
		// Parse existing YAML
		if err := yaml.Unmarshal(existingData, &config); err != nil {
			return fmt.Errorf("failed to parse existing .sops.yaml: %w", err)
		}

		// Check if rule already exists (idempotency)
		for _, rule := range config.CreationRules {
			if rule.PathRegex == pathRegex {
				fmt.Printf("⏭️  .sops.yaml already contains rule for %s\n", outputFile)
				return nil
			}
		}

		if force {
			// Force mode: replace all rules
			config.CreationRules = []CreationRule{
				{PathRegex: pathRegex, Age: publicKey},
			}
			fmt.Println("✅ Replaced .sops.yaml (force mode)")
		} else {
			// Append new rule
			config.CreationRules = append(config.CreationRules, CreationRule{
				PathRegex: pathRegex,
				Age:       publicKey,
			})
			fmt.Printf("✅ Added rule to .sops.yaml for %s\n", outputFile)
		}
	} else {
		// File doesn't exist, create new
		config.CreationRules = []CreationRule{
			{PathRegex: pathRegex, Age: publicKey},
		}
		fmt.Println("✅ Created .sops.yaml")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sopsYamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .sops.yaml: %w", err)
	}

	return nil
}
