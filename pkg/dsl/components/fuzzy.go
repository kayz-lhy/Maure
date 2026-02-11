package components

import (
	"fmt"
	"strings"
)

// FuzzyComponent 解析 field:term~1。
type FuzzyComponent struct{}

// TryParse 实现 FieldExprComponent。
func (FuzzyComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	if !strings.Contains(expr, "~") {
		return FieldParseResult{}, false, nil
	}
	if !strings.HasSuffix(expr, "~1") {
		return FieldParseResult{}, false, fmt.Errorf("only fuzzy distance ~1 is supported: %s", token)
	}
	term := strings.TrimSuffix(expr, "~1")
	if strings.TrimSpace(term) == "" {
		return FieldParseResult{}, false, fmt.Errorf("fuzzy term cannot be empty: %s", token)
	}
	return FieldParseResult{Kind: ExprFuzzy, Field: field, Text: term, Distance: 1}, true, nil
}
