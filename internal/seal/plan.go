package seal

import (
	"fmt"
	"strings"

	tools "github.com/YewFence/YewSeal/internal/doctor"
)

type codecPlan struct {
	userFormat     format
	sopsFormat     string
	needsRemarshal bool
	encryptAction  string
	decryptAction  string
	prepareEncrypt func([]byte) ([]byte, error)
	restoreDecrypt func([]byte) ([]byte, error)
}

func newCodecPlan(path, override string) (codecPlan, error) {
	if strings.TrimSpace(override) != "" {
		userFormat := parseFormat(override)
		if userFormat == formatUnknown {
			return codecPlan{}, fmt.Errorf("unsupported format override %q (supported: %s)", override, supportedFormats())
		}
		return codecPlanForFormat(userFormat), nil
	}

	userFormat := detectFormat(path)
	if userFormat == formatUnknown {
		return codecPlan{}, fmt.Errorf("could not detect format for %s (supported: %s). Hint: pass --format binary if this should be encrypted as a binary file", path, supportedFormats())
	}
	return codecPlanForFormat(userFormat), nil
}

func codecPlanForFormat(userFormat format) codecPlan {
	plan := codecPlan{
		userFormat:     userFormat,
		sopsFormat:     nativeSOPSFormat(userFormat),
		prepareEncrypt: passthrough,
		restoreDecrypt: passthrough,
	}

	if userFormat == formatTOML {
		plan.sopsFormat = "yaml"
		plan.needsRemarshal = true
		plan.encryptAction = "🔄 Converting TOML to YAML..."
		plan.decryptAction = "🔄 Converting YAML to TOML..."
		plan.prepareEncrypt = func(data []byte) ([]byte, error) {
			converted, err := tomlToYAML(data)
			if err != nil {
				return nil, fmt.Errorf("failed to convert TOML to YAML: %w", err)
			}
			return converted, nil
		}
		plan.restoreDecrypt = func(data []byte) ([]byte, error) {
			converted, err := yamlToTOML(data)
			if err != nil {
				return nil, fmt.Errorf("failed to convert YAML to TOML: %w", err)
			}
			return converted, nil
		}
	}

	return plan
}

func (p codecPlan) checkTools() error {
	if !p.needsRemarshal {
		return nil
	}
	return tools.CheckRemarshal()
}

func passthrough(data []byte) ([]byte, error) {
	return data, nil
}

func nativeSOPSFormat(userFormat format) string {
	switch userFormat {
	case formatYAML:
		return "yaml"
	case formatJSON:
		return "json"
	case formatENV:
		return "dotenv"
	case formatINI:
		return "ini"
	case formatBinary:
		return "binary"
	default:
		return ""
	}
}

func supportedFormats() string {
	return "toml, yaml, json, env, ini, binary"
}
