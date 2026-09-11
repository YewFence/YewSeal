package config

import "github.com/YewFence/YewSeal/internal/task"

// Selection policies keep discovery, authorization, and writes independent.
type selectionPolicy struct {
	discoveryMode        string
	scopeMode            string
	historicalRecipients bool
	writes               bool
}

func policyForCommand(command string) selectionPolicy {
	switch command {
	case task.ModeView:
		return selectionPolicy{discoveryMode: task.ModeDecrypt, scopeMode: task.ModeDecrypt, historicalRecipients: true}
	case task.ModeDiff:
		return selectionPolicy{discoveryMode: task.ModeDiff, scopeMode: task.ModeEncrypt, historicalRecipients: true}
	case task.ModePlan:
		return selectionPolicy{discoveryMode: task.ModePlan, scopeMode: task.ModePlan}
	case task.ModeDecrypt:
		return selectionPolicy{discoveryMode: task.ModeDecrypt, scopeMode: task.ModeDecrypt, historicalRecipients: true, writes: true}
	default:
		return selectionPolicy{discoveryMode: command, scopeMode: command, writes: true}
	}
}
