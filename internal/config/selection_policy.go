package config

import "github.com/YewFence/YewSeal/internal/task"

// Selection policies keep discovery, authorization, and writes independent.
type selectionPolicy struct {
	discoveryMode        string
	scopeMode            string
	historicalRecipients bool
	writes               bool
	allowEmptySelection  bool
}

func policyForCommand(command string) selectionPolicy {
	switch command {
	case task.ModeView:
		return selectionPolicy{discoveryMode: task.ModeDecrypt, scopeMode: task.ModeDecrypt, historicalRecipients: true}
	case task.ModeEncrypt:
		return selectionPolicy{discoveryMode: task.ModeEncrypt, scopeMode: task.ModeEncrypt, writes: true, allowEmptySelection: true}
	case task.ModeDiff:
		return selectionPolicy{discoveryMode: task.ModeDiff, scopeMode: task.ModeEncrypt, historicalRecipients: true, allowEmptySelection: true}
	case task.ModePlan:
		return selectionPolicy{discoveryMode: task.ModePlan, scopeMode: task.ModePlan, allowEmptySelection: true}
	case task.ModeClean:
		return selectionPolicy{discoveryMode: task.ModeClean, scopeMode: task.ModeEncrypt, historicalRecipients: true, writes: true, allowEmptySelection: true}
	case task.ModeVerify:
		// Verify is directionless like plan: matches either side, reads but never writes.
		return selectionPolicy{discoveryMode: task.ModePlan, scopeMode: task.ModePlan, allowEmptySelection: true}
	case task.ModeDecrypt:
		return selectionPolicy{discoveryMode: task.ModeDecrypt, scopeMode: task.ModeDecrypt, historicalRecipients: true, writes: true, allowEmptySelection: true}
	default:
		return selectionPolicy{discoveryMode: command, scopeMode: command, writes: true}
	}
}
