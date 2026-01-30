package crypto

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// YAMLToTOML converts YAML content to TOML format
func YAMLToTOML(yamlContent []byte) ([]byte, error) {
	// Parse YAML
	var data interface{}
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert yaml.v3 data structure to something toml can handle
	converted := convertYAMLtoMap(data)

	// Encode to TOML without indentation
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	encoder.Indent = "" // Remove indentation to follow TOML standard format
	if err := encoder.Encode(converted); err != nil {
		return nil, fmt.Errorf("failed to encode TOML: %w", err)
	}

	return buf.Bytes(), nil
}

// TOMLToYAML converts TOML content to YAML format
func TOMLToYAML(tomlContent []byte) ([]byte, error) {
	// Parse TOML
	var data interface{}
	if err := toml.Unmarshal(tomlContent, &data); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Encode to YAML
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}

	return yamlBytes, nil
}

// convertYAMLtoMap recursively converts yaml.v3 types to basic Go types
func convertYAMLtoMap(input interface{}) interface{} {
	switch v := input.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertYAMLtoMap(value)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			strKey := fmt.Sprintf("%v", key)
			result[strKey] = convertYAMLtoMap(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = convertYAMLtoMap(value)
		}
		return result
	default:
		return v
	}
}
