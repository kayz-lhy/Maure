package dsl

import "strings"

// ParsedQuery 是 DSL 解析结果，表达式与可选元信息分离。
type ParsedQuery struct {
	Version int
	// RequireIn 表示查询声明了强制 IN 约束。
	RequireIn bool
	Scopes    []Scope
	Expr      Expr
	Limit     *LimitClause
	Sort      []SortClause
}

// Scope 表示查询作用域，例如 index("app")。
type Scope struct {
	Kind  string
	Value string
}

// LimitClause 表示分页子句。
type LimitClause struct {
	From int
	Size int
}

// SortClause 表示排序子句。
type SortClause struct {
	Field string
	Desc  bool
}

// Expr 是 DSL 表达式节点。
type Expr interface {
	exprNode()
}

// TermExpr 是词项表达式。
type TermExpr struct {
	Field string
	Value string
}

func (TermExpr) exprNode() {}

// PhraseExpr 是短语表达式。
type PhraseExpr struct {
	Field string
	Text  string
}

func (PhraseExpr) exprNode() {}

// RangeValueKind 描述范围类型。
type RangeValueKind int

const (
	RangeValueNumber RangeValueKind = iota
	RangeValueTime
)

// RangeExpr 是范围表达式。
type RangeExpr struct {
	Field     string
	Lower     string
	Upper     string
	Kind      RangeValueKind
	Inclusive bool
}

func (RangeExpr) exprNode() {}

// WildcardExpr 是通配符表达式（prefix-only）。
type WildcardExpr struct {
	Field  string
	Prefix string
}

func (WildcardExpr) exprNode() {}

// FuzzyExpr 是模糊表达式。
type FuzzyExpr struct {
	Field    string
	Term     string
	Distance int
}

func (FuzzyExpr) exprNode() {}

// ExistsExpr 是字段存在表达式，例如 field:* 。
type ExistsExpr struct {
	Field string
}

func (ExistsExpr) exprNode() {}

// AndExpr 是布尔与表达式。
type AndExpr struct {
	Left  Expr
	Right Expr
}

func (AndExpr) exprNode() {}

// OrExpr 是布尔或表达式。
type OrExpr struct {
	Left  Expr
	Right Expr
}

func (OrExpr) exprNode() {}

// NotExpr 是一元否定表达式。
type NotExpr struct {
	Sub Expr
}

func (NotExpr) exprNode() {}

// FilterNotExpr 表示 A NOT B（include + exclude）。
type FilterNotExpr struct {
	Include Expr
	Exclude Expr
}

func (FilterNotExpr) exprNode() {}

func normalizeKeyword(token string) string {
	upper := strings.ToUpper(token)
	if upper == "AND" || upper == "OR" || upper == "NOT" || upper == "IN" || upper == "LIMIT" || upper == "SORT" || upper == "BY" || upper == "ASC" || upper == "DESC" || upper == "TO" || upper == "REQUIRE_IN" {
		return upper
	}
	return token
}
