package query

import (
	"fmt"
	"maure/pkg/dsl"
	"strings"
)

type exprCompiler func(expr dsl.Expr) (Query, error)

// DSLAdapter 将 DSL Plan 适配为 QueryPlan。
type DSLAdapter struct {
	compilers map[string]exprCompiler
}

// NewDSLAdapter 创建适配器。
func NewDSLAdapter() *DSLAdapter {
	a := &DSLAdapter{compilers: make(map[string]exprCompiler, 9)}
	a.register("dsl.TermExpr", a.compileTerm)
	a.register("dsl.PhraseExpr", a.compilePhrase)
	a.register("dsl.RangeExpr", a.compileRange)
	a.register("dsl.WildcardExpr", a.compileWildcard)
	a.register("dsl.FuzzyExpr", a.compileFuzzy)
	a.register("dsl.ExistsExpr", a.compileExists)
	a.register("dsl.AndExpr", a.compileAnd)
	a.register("dsl.OrExpr", a.compileOr)
	a.register("dsl.NotExpr", a.compileNot)
	a.register("dsl.FilterNotExpr", a.compileFilterNot)
	return a
}

func (a *DSLAdapter) register(kind string, fn exprCompiler) {
	a.compilers[kind] = fn
}

// Compile 实现 dsl.Compiler，返回 QueryPlan。
func (a *DSLAdapter) Compile(plan *dsl.Plan) (dsl.Executable, error) {
	if plan == nil {
		return &QueryPlan{}, nil
	}
	compiled, err := a.compileExpr(plan.ExprTree)
	if err != nil {
		return nil, err
	}
	return &QueryPlan{
		Version:   plan.Version,
		RequireIn: plan.RequireIn,
		Scopes:    plan.Scopes,
		Limit:     plan.Limit,
		Sort:      plan.Sort,
		Query:     compiled,
	}, nil
}

func (a *DSLAdapter) compileExpr(expr dsl.Expr) (Query, error) {
	if expr == nil {
		return nil, nil
	}
	kind := fmt.Sprintf("%T", expr)
	fn, ok := a.compilers[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported DSL expression: %s", kind)
	}
	return fn(expr)
}

func (a *DSLAdapter) compileTerm(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.TermExpr)
	if e.Field == "" {
		return NewTermQuery(e.Value), nil
	}
	return NewTermQuery(e.Value).WithField(e.Field), nil
}

func (a *DSLAdapter) compilePhrase(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.PhraseExpr)
	terms := strings.Fields(e.Text)
	if len(terms) == 0 {
		return nil, nil
	}
	if len(terms) == 1 {
		q := NewTermQuery(terms[0])
		if e.Field != "" {
			q.WithField(e.Field)
		}
		return q, nil
	}
	q := NewPhraseQuery(terms...)
	if e.Field != "" {
		q.WithField(e.Field)
	}
	return q, nil
}

func (a *DSLAdapter) compileRange(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.RangeExpr)
	kind := RangeValueNumber
	if e.Kind == dsl.RangeValueTime {
		kind = RangeValueTime
	}
	return NewRangeQuery(e.Field, e.Lower, e.Upper, kind, e.Inclusive), nil
}

func (a *DSLAdapter) compileWildcard(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.WildcardExpr)
	return NewWildcardQuery(e.Field, e.Prefix), nil
}

func (a *DSLAdapter) compileFuzzy(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.FuzzyExpr)
	return NewFuzzyQuery(e.Field, e.Term, e.Distance), nil
}

func (a *DSLAdapter) compileExists(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.ExistsExpr)
	return NewExistsQuery(e.Field), nil
}

func (a *DSLAdapter) compileAnd(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.AndExpr)
	left, err := a.compileExpr(e.Left)
	if err != nil {
		return nil, err
	}
	right, err := a.compileExpr(e.Right)
	if err != nil {
		return nil, err
	}
	return NewConjunctionQuery(left, right), nil
}

func (a *DSLAdapter) compileOr(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.OrExpr)
	left, err := a.compileExpr(e.Left)
	if err != nil {
		return nil, err
	}
	right, err := a.compileExpr(e.Right)
	if err != nil {
		return nil, err
	}
	return NewDisjunctionQuery(left, right), nil
}

func (a *DSLAdapter) compileNot(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.NotExpr)
	sub, err := a.compileExpr(e.Sub)
	if err != nil {
		return nil, err
	}
	return &notQuery{subQuery: sub}, nil
}

func (a *DSLAdapter) compileFilterNot(expr dsl.Expr) (Query, error) {
	e := expr.(dsl.FilterNotExpr)
	include, err := a.compileExpr(e.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := a.compileExpr(e.Exclude)
	if err != nil {
		return nil, err
	}
	boolQuery := NewBooleanQuery()
	boolQuery.Add(include, OccurMust, 1.0)
	boolQuery.Add(exclude, OccurMustNot, 1.0)
	return boolQuery, nil
}
