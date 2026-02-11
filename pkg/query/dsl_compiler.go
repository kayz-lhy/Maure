package query

import "maure/pkg/dsl"

// QueryPlan 是 DSL 到执行层的中间结果。
type QueryPlan struct {
	Version   int
	RequireIn bool
	Scopes    []dsl.Scope
	Limit     *dsl.LimitClause
	Sort      []dsl.SortClause
	Query     Query
}

// CompileDSL 将 DSL AST 编译为可执行 QueryPlan。
func CompileDSL(parsed *dsl.ParsedQuery) (*QueryPlan, error) {
	plan := &dsl.Plan{}
	if parsed != nil {
		plan.Version = parsed.Version
		plan.RequireIn = parsed.RequireIn
		plan.ExprTree = parsed.Expr
		plan.Scopes = parsed.Scopes
		plan.Limit = parsed.Limit
		plan.Sort = parsed.Sort
	}
	exec, err := NewDSLAdapter().Compile(plan)
	if err != nil {
		return nil, err
	}
	return exec.(*QueryPlan), nil
}
