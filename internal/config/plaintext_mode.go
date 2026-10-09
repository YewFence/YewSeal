package config

import "fmt"

const (
	PlaintextInplace  = "inplace"
	PlaintextDelivery = "delivery"
)

func normalizePlaintextMode(mode string) (string, error) {
	switch mode {
	case "", PlaintextInplace:
		return PlaintextInplace, nil
	case PlaintextDelivery:
		return PlaintextDelivery, nil
	default:
		return "", fmt.Errorf("plaintext_mode must be inplace or delivery, got %q", mode)
	}
}
