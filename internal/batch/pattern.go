package batch

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PatternRule struct {
	Raw           string
	Negated       bool
	DirectoryOnly bool
	Anchored      bool
	HasSlash      bool
	Segments      []string
}

type PatternMatcher struct {
	rules []PatternRule
}

func NewPatternMatcher(patterns []string) (PatternMatcher, error) {
	rules, err := ParsePatternRules(patterns)
	if err != nil {
		return PatternMatcher{}, err
	}
	return PatternMatcher{rules: rules}, nil
}

func ParsePatternRules(patterns []string) ([]PatternRule, error) {
	rules := make([]PatternRule, 0, len(patterns))
	for _, raw := range patterns {
		ruleText := strings.TrimSpace(raw)
		if ruleText == "" || strings.HasPrefix(ruleText, "#") {
			continue
		}
		if strings.HasPrefix(ruleText, `\#`) {
			ruleText = ruleText[1:]
		}

		rule := PatternRule{Raw: raw}
		if strings.HasPrefix(ruleText, "!") {
			rule.Negated = true
			ruleText = strings.TrimSpace(ruleText[1:])
			if ruleText == "" {
				return nil, fmt.Errorf("invalid scan pattern %q: missing pattern after negation", raw)
			}
		}
		ruleText = filepath.ToSlash(ruleText)
		if strings.HasSuffix(ruleText, "/") {
			rule.DirectoryOnly = true
			ruleText = strings.TrimRight(ruleText, "/")
			if ruleText == "" {
				return nil, fmt.Errorf("invalid scan pattern %q: directory pattern is empty", raw)
			}
		}
		if strings.HasPrefix(ruleText, "/") {
			rule.Anchored = true
			ruleText = strings.TrimLeft(ruleText, "/")
			if ruleText == "" {
				return nil, fmt.Errorf("invalid scan pattern %q: anchored pattern is empty", raw)
			}
		}

		rule.HasSlash = strings.Contains(ruleText, "/")
		rule.Segments = strings.Split(ruleText, "/")
		for _, segment := range rule.Segments {
			if segment == "" {
				return nil, fmt.Errorf("invalid scan pattern %q: empty path segment", raw)
			}
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (m PatternMatcher) Decision(path string, isDir bool) (bool, bool) {
	normalized := normalizeScanPath(path)
	if normalized == "" {
		return false, false
	}

	decided := false
	included := false
	for _, rule := range m.rules {
		if !rule.matches(normalized, isDir) {
			continue
		}
		decided = true
		included = !rule.Negated
	}
	return decided, included
}

func (r PatternRule) matches(path string, isDir bool) bool {
	if r.DirectoryOnly && !isDir {
		return false
	}

	pathSegments := strings.Split(path, "/")
	if r.Anchored || r.HasSlash {
		return matchSegments(r.Segments, pathSegments)
	}

	for _, segment := range pathSegments {
		if matchSegment(r.Segments[0], segment) {
			return true
		}
	}
	return false
}

func normalizeScanPath(path string) string {
	normalized := filepath.ToSlash(path)
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}

func matchSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], path) {
			return true
		}
		for i := range path {
			if matchSegments(pattern[1:], path[i+1:]) {
				return true
			}
		}
		return false
	}
	if len(path) == 0 {
		return false
	}
	return matchSegment(pattern[0], path[0]) && matchSegments(pattern[1:], path[1:])
}

func matchSegment(pattern, value string) bool {
	return matchSegmentFrom(pattern, value, 0, 0)
}

func matchSegmentFrom(pattern, value string, patternIndex, valueIndex int) bool {
	for patternIndex < len(pattern) {
		switch pattern[patternIndex] {
		case '*':
			for nextValue := valueIndex; nextValue <= len(value); nextValue++ {
				if matchSegmentFrom(pattern, value, patternIndex+1, nextValue) {
					return true
				}
			}
			return false
		case '?':
			if valueIndex >= len(value) {
				return false
			}
			patternIndex++
			valueIndex++
		default:
			if valueIndex >= len(value) || pattern[patternIndex] != value[valueIndex] {
				return false
			}
			patternIndex++
			valueIndex++
		}
	}
	return valueIndex == len(value)
}
