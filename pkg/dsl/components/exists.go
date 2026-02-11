package components

// ExistsComponent 解析 field:*。
type ExistsComponent struct{}

// TryParse 实现 FieldExprComponent。
func (ExistsComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	_ = token
	if expr != "*" {
		return FieldParseResult{}, false, nil
	}
	return FieldParseResult{Kind: ExprExists, Field: field}, true, nil
}
