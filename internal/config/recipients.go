package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"filippo.io/age"
)

var recipientAliasPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// RecipientProvenance describes where a resolved authorization set came from.
type RecipientProvenance struct {
	Kind       string
	ConfigPath string
	Aliases    []string

	// RegistrySources maps each alias used by the set to its defining config file.
	RegistrySources map[string]string

	// DeclarationSource identifies the file/group/defaults declaration.
	DeclarationSource ValueSource

	// EffectiveSource identifies the winning authorization layer.
	EffectiveSource ValueSource
}

// ResolvedRecipients is the canonical public authorization for one file.
type ResolvedRecipients struct {
	Aliases    []string
	Recipients []string
	Provenance RecipientProvenance
}

// ValidateRecipientConfig validates the registry and the optional defaults.
func (c *Config) ValidateRecipientConfig() error {
	aliases := make([]string, 0, len(c.Recipients.Registry))
	for alias := range c.Recipients.Registry {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		recipient := c.Recipients.Registry[alias]
		if !recipientAliasPattern.MatchString(alias) {
			return fmt.Errorf("invalid recipient alias %q", alias)
		}
		if strings.TrimSpace(recipient) == "" {
			return fmt.Errorf("recipient alias %q has an empty public key", alias)
		}
		if _, err := age.ParseX25519Recipient(strings.TrimSpace(recipient)); err != nil {
			return fmt.Errorf("recipient alias %q has invalid Age recipient: %w", alias, err)
		}
	}
	if c.Recipients.Defaults != nil {
		if len(*c.Recipients.Defaults) == 0 {
			return fmt.Errorf("recipients.defaults must not be empty")
		}
	}
	seen := make(map[string]string, len(c.Recipients.Registry))
	for _, alias := range aliases {
		canonical := strings.TrimSpace(c.Recipients.Registry[alias])
		if previous, ok := seen[canonical]; ok {
			return fmt.Errorf("recipient aliases %q and %q resolve to the same public key", previous, alias)
		}
		seen[canonical] = alias
	}
	return nil
}

// ResolveFileRecipients resolves the winning file, group, or defaults authorization set.
func (c *Config) ResolveFileRecipients(pair FilePair) (ResolvedRecipients, error) {
	if pair.Recipients != nil {
		if len(*pair.Recipients) == 0 {
			return ResolvedRecipients{}, fmt.Errorf("file %q has an empty recipient set", pair.PlaintextPath)
		}
		kind := "file"
		if pair.Source == PairSourceScan {
			kind = "group"
		}
		source := pair.RecipientSource
		if source.Kind == "" {
			source = ValueSource{Kind: kind, ConfigPath: pair.ConfigPath, Detail: "recipients"}
		}
		if source.Kind != "" {
			kind = source.Kind
		}
		return c.resolveAliasesWithSource(*pair.Recipients, kind, source)
	}
	if c.Recipients.Defaults == nil {
		return ResolvedRecipients{}, fmt.Errorf("file %q has no recipient set", pair.PlaintextPath)
	}
	source := ValueSource{Kind: "defaults", ConfigPath: c.Recipients.DefaultsConfigPath, Detail: "recipients.defaults"}
	return c.resolveAliasesWithSource(*c.Recipients.Defaults, "defaults", source)
}

func (c *Config) resolveAliasesWithSource(aliases []string, kind string, source ValueSource) (ResolvedRecipients, error) {
	recipients, err := c.resolveAliases(aliases)
	if err != nil {
		return ResolvedRecipients{}, err
	}
	registrySources := make(map[string]string, len(aliases))
	for _, alias := range aliases {
		if source := c.Recipients.RegistrySources[alias]; source != "" {
			registrySources[alias] = source
		}
	}
	provenance := RecipientProvenance{
		Kind:              kind,
		ConfigPath:        source.ConfigPath,
		Aliases:           append([]string(nil), aliases...),
		RegistrySources:   registrySources,
		DeclarationSource: source,
		EffectiveSource:   source,
	}
	return ResolvedRecipients{
		Aliases:    append([]string(nil), aliases...),
		Recipients: recipients,
		Provenance: provenance,
	}, nil
}

func (c *Config) resolveAliases(aliases []string) ([]string, error) {
	seenAlias := make(map[string]struct{}, len(aliases))
	recipients := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		if _, err := age.ParseX25519Recipient(strings.TrimSpace(alias)); err == nil {
			return nil, fmt.Errorf("raw Age recipient %q is not supported; define it in recipients.registry and reference its alias", alias)
		}
		if !recipientAliasPattern.MatchString(alias) {
			return nil, fmt.Errorf("invalid recipient alias %q", alias)
		}
		if _, ok := seenAlias[alias]; ok {
			return nil, fmt.Errorf("duplicate recipient alias %q", alias)
		}
		seenAlias[alias] = struct{}{}
		recipient, ok := c.Recipients.Registry[alias]
		if !ok {
			return nil, fmt.Errorf("unknown recipient alias %q", alias)
		}
		canonical := strings.TrimSpace(recipient)
		if _, err := age.ParseX25519Recipient(canonical); err != nil {
			return nil, fmt.Errorf("recipient alias %q has invalid Age recipient: %w", alias, err)
		}
		recipients = append(recipients, canonical)
	}
	sort.Strings(recipients)
	return recipients, nil
}
func cloneOptionalStrings(values *[]string) *[]string {
	if values == nil {
		return nil
	}
	return cloneStringSlicePtr(*values)
}

func cloneStringSlicePtr(values []string) *[]string {
	if values == nil {
		return nil
	}
	clone := append([]string(nil), values...)
	return &clone
}
