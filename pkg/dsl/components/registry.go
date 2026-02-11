package components

// ExprKind 描述字段表达式类型。
type ExprKind string

const (
	ExprExists   ExprKind = "exists"
	ExprPhrase   ExprKind = "phrase"
	ExprRange    ExprKind = "range"
	ExprWildcard ExprKind = "wildcard"
	ExprFuzzy    ExprKind = "fuzzy"
	ExprTerm     ExprKind = "term"
)

// FieldParseResult 是字段表达式组件的统一返回。
type FieldParseResult struct {
	Kind      ExprKind
	Field     string
	Text      string
	Lower     string
	Upper     string
	ValueKind string
	Inclusive bool
	Distance  int
}

// FieldExprComponent 负责解析 field:expr 形式语句。
type FieldExprComponent interface {
	TryParse(field string, expr string, token string) (FieldParseResult, bool, error)
}

// Registry 保存 field expression 组件列表。
type Registry struct {
	fieldExpr []FieldExprComponent
}

// NewRegistry 创建组件注册表。
func NewRegistry() *Registry {
	return &Registry{fieldExpr: make([]FieldExprComponent, 0, 8)}
}

// RegisterFieldExpr 注册字段表达式组件。
func (r *Registry) RegisterFieldExpr(c FieldExprComponent) {
	r.fieldExpr = append(r.fieldExpr, c)
}

// ParseFieldExpression 按注册顺序尝试组件解析。
func (r *Registry) ParseFieldExpression(field string, expr string, token string) (FieldParseResult, bool, error) {
	for _, c := range r.fieldExpr {
		result, ok, err := c.TryParse(field, expr, token)
		if err != nil {
			return FieldParseResult{}, false, err
		}
		if ok {
			return result, true, nil
		}
	}
	return FieldParseResult{}, false, nil
}

// DefaultRegistry 返回 DSL V1 默认组件组合。
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.RegisterFieldExpr(ExistsComponent{})
	r.RegisterFieldExpr(PhraseComponent{})
	r.RegisterFieldExpr(RangeComponent{})
	r.RegisterFieldExpr(WildcardComponent{})
	r.RegisterFieldExpr(FuzzyComponent{})
	r.RegisterFieldExpr(TermComponent{})
	return r
}
