package components

import (
	"fmt"
	"strings"
)

// WildcardComponent 解析 field:prefix*。
type WildcardComponent struct{}

// TryParse 实现 FieldExprComponent。
func (WildcardComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	if strings.Contains(expr, "?") {
		return FieldParseResult{}, false, fmt.Errorf("wildcard '?' is not supported: %s", token)
	}
	if !strings.Contains(expr, "*") {
		return FieldParseResult{}, false, nil
	}
	if !strings.HasSuffix(expr, "*") || strings.Count(expr, "*") != 1 {
		return FieldParseResult{}, false, fmt.Errorf("only suffix '*' wildcard is supported: %s", token)
	}
	prefix := strings.TrimSuffix(expr, "*")
	if strings.TrimSpace(prefix) == "" {
		return FieldParseResult{}, false, fmt.Errorf("wildcard prefix cannot be empty: %s", token)
	}
	return FieldParseResult{Kind: ExprWildcard, Field: field, Text: prefix}, true, nil
}
