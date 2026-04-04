package project

import (
	"fmt"
	"os"
	"regexp"

	"github.com/YewFence/YewSeal/internal/config"
	"gopkg.in/yaml.v3"
)

const sopsYamlPath = ".sops.yaml"

// SopsConfig represents the structure of .sops.yaml
type SopsConfig struct {
	CreationRules []CreationRule `yaml:"creation_rules"`
}

// CreationRule represents a single rule in .sops.yaml
type CreationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

// UpdateSopsYaml creates or updates .sops.yaml with proper YAML handling.
func UpdateSopsYaml(outputFile, publicKey string, force bool) error {
	pathRegex := buildPathRegex(outputFile)

	currentConfig, err := loadSopsConfig()
	if err != nil {
		return err
	}
	if currentConfig == nil {
		currentConfig = &SopsConfig{}
	}

	for _, rule := range currentConfig.CreationRules {
		if rule.PathRegex == pathRegex {
			fmt.Printf("⏭️  .sops.yaml already contains rule for %s\n", outputFile)
			return nil
		}
	}

	if force {
		currentConfig.CreationRules = []CreationRule{
			{PathRegex: pathRegex, Age: publicKey},
		}
		fmt.Println("✅ Replaced .sops.yaml (force mode)")
	} else {
		currentConfig.CreationRules = append(currentConfig.CreationRules, CreationRule{
			PathRegex: pathRegex,
			Age:       publicKey,
		})
		if len(currentConfig.CreationRules) == 1 {
			fmt.Println("✅ Created .sops.yaml")
		} else {
			fmt.Printf("✅ Added rule to .sops.yaml for %s\n", outputFile)
		}
	}

	return writeSopsConfig(*currentConfig)
}

// SyncSopsYaml rewrites .sops.yaml from configured encrypted files.
func SyncSopsYaml(filePairs []config.FilePair, publicKey string) error {
	creationRules := make([]CreationRule, 0, len(filePairs))
	seen := make(map[string]struct{}, len(filePairs))

	for _, filePair := range filePairs {
		if filePair.EncryptedPath == "" {
			continue
		}

		pathRegex := buildPathRegex(filePair.EncryptedPath)
		if _, ok := seen[pathRegex]; ok {
			continue
		}
		seen[pathRegex] = struct{}{}
		creationRules = append(creationRules, CreationRule{
			PathRegex: pathRegex,
			Age:       publicKey,
		})
	}

	if err := writeSopsConfig(SopsConfig{CreationRules: creationRules}); err != nil {
		return err
	}

	fmt.Printf("✅ Synced .sops.yaml with %d rule(s)\n", len(creationRules))
	return nil
}

func loadSopsConfig() (*SopsConfig, error) {
	existingData, err := os.ReadFile(sopsYamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read existing .sops.yaml: %w", err)
	}

	var currentConfig SopsConfig
	if err := yaml.Unmarshal(existingData, &currentConfig); err != nil {
		return nil, fmt.Errorf("failed to parse existing .sops.yaml: %w", err)
	}

	return &currentConfig, nil
}

func writeSopsConfig(config SopsConfig) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(sopsYamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .sops.yaml: %w", err)
	}

	return nil
}

func buildPathRegex(outputFile string) string {
	escapedOutput := regexp.QuoteMeta(outputFile)
	return "^" + escapedOutput + "$"
}
