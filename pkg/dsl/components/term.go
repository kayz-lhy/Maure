package components

import "strings"

// TermComponent 解析兜底 field:value。
type TermComponent struct{}

// TryParse 实现 FieldExprComponent。
func (TermComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	_ = token
	return FieldParseResult{Kind: ExprTerm, Field: field, Text: strings.ToLower(expr)}, true, nil
}
