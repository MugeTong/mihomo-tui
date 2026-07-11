package rules

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed default.yaml
var defaultYAML string

type Rule struct {
	Type    string
	Value   string
	Policy  string
	Options []string
}

func Default() ([]Rule, error) {
	return parse(defaultYAML)
}

func DefaultYAML() string {
	return defaultYAML
}

func parse(source string) ([]Rule, error) {
	lines := strings.Split(source, "\n")
	rules := make([]Rule, 0, len(lines))
	insideRules := false

	for lineNumber, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "rules:" {
			insideRules = true
			continue
		}
		if !insideRules || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "- ") {
			return nil, fmt.Errorf("default rules line %d: expected list item", lineNumber+1)
		}

		parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "- ")), ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) < 2 {
			return nil, fmt.Errorf("default rules line %d: incomplete rule", lineNumber+1)
		}

		rule := Rule{Type: parts[0]}
		if rule.Type == "MATCH" {
			rule.Policy = parts[1]
			rule.Value = "-"
			rule.Options = append([]string(nil), parts[2:]...)
		} else {
			if len(parts) < 3 {
				return nil, fmt.Errorf("default rules line %d: missing policy", lineNumber+1)
			}
			rule.Value = parts[1]
			rule.Policy = parts[2]
			rule.Options = append([]string(nil), parts[3:]...)
		}
		rules = append(rules, rule)
	}

	if len(rules) == 0 {
		return nil, fmt.Errorf("embedded default rule set is empty")
	}
	return rules, nil
}
