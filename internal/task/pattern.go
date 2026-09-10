package task

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// PatternRule 是一条通过校验的 gitignore 风格规则，匹配语义由 go-git 的
// gitignore 实现提供。YewSeal 的 patterns 表达“选中”而非“忽略”：普通规则
// 命中表示选中文件，`!` 规则命中表示排除文件。
type PatternRule struct {
	Raw     string
	Negated bool
	pattern gitignore.Pattern
}

// PatternMatcher 按顺序应用一组 PatternRule，最后一条命中的规则生效。
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

// ParsePatternRules 校验并编译一组 gitignore 风格规则；空行和 # 开头的行
// 会被跳过，`\#` 开头的规则按字面量 # 处理。
func ParsePatternRules(patterns []string) ([]PatternRule, error) {
	rules := make([]PatternRule, 0, len(patterns))
	for _, raw := range patterns {
		ruleText := raw
		if ruleText == "" || strings.HasPrefix(ruleText, "#") {
			continue
		}
		if strings.HasPrefix(ruleText, `\#`) {
			ruleText = ruleText[1:]
		}

		rule := PatternRule{Raw: raw}
		// 匹配语义遵循上游 gitignore 与 Go filepath.Match：空白、转义与
		// 路径分隔符都不做额外归一化（patterns 在所有平台以 `/` 作为
		// 路径分隔符），本层只做结构校验。
		check := ruleText
		if strings.HasPrefix(check, "!") {
			rule.Negated = true
			check = check[1:]
			if check == "" {
				return nil, fmt.Errorf("invalid group pattern %q: missing pattern after negation", raw)
			}
		}
		if strings.HasSuffix(check, "/") {
			check = strings.TrimRight(check, "/")
			if check == "" {
				return nil, fmt.Errorf("invalid group pattern %q: directory pattern is empty", raw)
			}
		}
		if strings.HasPrefix(check, "/") {
			check = strings.TrimLeft(check, "/")
			if check == "" {
				return nil, fmt.Errorf("invalid group pattern %q: anchored pattern is empty", raw)
			}
		}
		for _, segment := range strings.Split(check, "/") {
			if segment == "" {
				return nil, fmt.Errorf("invalid group pattern %q: empty path segment", raw)
			}
		}

		rule.pattern = gitignore.ParsePattern(ruleText, nil)
		rules = append(rules, rule)
	}
	return rules, nil
}

// Decision 报告路径是否被任何规则命中（decided），以及按最后命中规则
// 计算的是否选中（included）。
func (m PatternMatcher) Decision(path string, isDir bool) (bool, bool) {
	normalized := normalizePatternPath(path)
	if normalized == "" {
		return false, false
	}
	segments := strings.Split(normalized, "/")

	decided := false
	included := false
	for _, rule := range m.rules {
		switch rule.pattern.Match(segments, isDir) {
		case gitignore.Exclude:
			// gitignore 的“忽略”对应 YewSeal 的“选中”。
			decided, included = true, true
		case gitignore.Include:
			decided, included = true, false
		}
	}
	return decided, included
}

func normalizePatternPath(path string) string {
	normalized := filepath.ToSlash(path)
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.TrimPrefix(normalized, "/")
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}
