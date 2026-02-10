package query

import (
	"fmt"
	"strings"

	"maure/pkg/dsl"
)

// QueryPlan 是 DSL 到执行层的中间结果。
type QueryPlan struct {
	Version int
	Scopes  []dsl.Scope
	Limit   *dsl.LimitClause
	Sort    []dsl.SortClause
	Query   Query
}

// CompileDSL 将 DSL AST 编译为可执行 QueryPlan。
func CompileDSL(parsed *dsl.ParsedQuery) (*QueryPlan, error) {
	if parsed == nil {
		return &QueryPlan{}, nil
	}
	compiled, err := compileExpr(parsed.Expr)
	if err != nil {
		return nil, err
	}
	return &QueryPlan{
		Version: parsed.Version,
		Scopes:  parsed.Scopes,
		Limit:   parsed.Limit,
		Sort:    parsed.Sort,
		Query:   compiled,
	}, nil
}

func compileExpr(expr dsl.Expr) (Query, error) {
	switch e := expr.(type) {
	case nil:
		return nil, nil
	case dsl.TermExpr:
		if e.Field == "" {
			return NewTermQuery(e.Value), nil
		}
		// 当前倒排执行层暂未对 TermQuery 做字段过滤，这里保留字段元信息，后续可扩展。
		return NewTermQuery(e.Value).WithField(e.Field), nil
	case dsl.PhraseExpr:
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
		return NewPhraseQuery(terms...), nil
	case dsl.RangeExpr:
		kind := RangeValueNumber
		if e.Kind == dsl.RangeValueTime {
			kind = RangeValueTime
		}
		return NewRangeQuery(e.Field, e.Lower, e.Upper, kind, e.Inclusive), nil
	case dsl.WildcardExpr:
		return NewWildcardQuery(e.Field, e.Prefix), nil
	case dsl.FuzzyExpr:
		return NewFuzzyQuery(e.Field, e.Term, e.Distance), nil
	case dsl.AndExpr:
		left, err := compileExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return NewConjunctionQuery(left, right), nil
	case dsl.OrExpr:
		left, err := compileExpr(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := compileExpr(e.Right)
		if err != nil {
			return nil, err
		}
		return NewDisjunctionQuery(left, right), nil
	case dsl.NotExpr:
		sub, err := compileExpr(e.Sub)
		if err != nil {
			return nil, err
		}
		return &notQuery{subQuery: sub}, nil
	case dsl.FilterNotExpr:
		include, err := compileExpr(e.Include)
		if err != nil {
			return nil, err
		}
		exclude, err := compileExpr(e.Exclude)
		if err != nil {
			return nil, err
		}
		boolQuery := NewBooleanQuery()
		boolQuery.Add(include, OccurMust, 1.0)
		boolQuery.Add(exclude, OccurMustNot, 1.0)
		return boolQuery, nil
	default:
		return nil, fmt.Errorf("unsupported DSL expression: %T", e)
	}
}
