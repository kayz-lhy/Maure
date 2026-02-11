package components

import "strings"

// PhraseComponent 解析 field:"..."。
type PhraseComponent struct{}

// TryParse 实现 FieldExprComponent。
func (PhraseComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	_ = token
	if len(expr) < 2 || !strings.HasPrefix(expr, `"`) || !strings.HasSuffix(expr, `"`) {
		return FieldParseResult{}, false, nil
	}
	text := expr[1 : len(expr)-1]
	return FieldParseResult{Kind: ExprPhrase, Field: field, Text: text}, true, nil
}
