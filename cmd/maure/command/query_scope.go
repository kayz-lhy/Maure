package command

import (
	"fmt"
	"strings"

	"maure/pkg/dsl"
	"maure/pkg/query"
)

// applyScopeQuery 将 DSL 作用域转成可执行过滤条件。
//
// 当前仅支持 index()：
// 1. 若文档包含 index 字段：按 index 字段过滤（多个 scope 为 OR）。
// 2. 若文档不包含 index 字段：视为单索引路由信息，不做过滤。
func applyScopeQuery(base query.Query, scopes []dsl.Scope, hasIndexField bool, forceIn bool) (query.Query, error) {
	if forceIn && len(scopes) == 0 {
		return nil, fmt.Errorf("已启用强制 IN，请在查询中显式指定 IN index(\"...\")")
	}
	if len(scopes) == 0 {
		return base, nil
	}

	for _, scope := range scopes {
		if !strings.EqualFold(scope.Kind, "index") {
			return nil, fmt.Errorf("作用域 %s 尚未支持", scope.Kind)
		}
	}

	if !hasIndexField {
		if forceIn {
			return nil, fmt.Errorf("已启用强制 IN，但文档中不存在 index 字段，无法执行作用域过滤")
		}
		return base, nil
	}

	indexTerms := make([]query.Query, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		v := strings.TrimSpace(scope.Value)
		if v == "" {
			return nil, fmt.Errorf("index 作用域值不能为空")
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		indexTerms = append(indexTerms, query.NewTermQuery(v).WithField("index"))
	}
	if len(indexTerms) == 0 {
		return nil, fmt.Errorf("index 作用域值不能为空")
	}

	var scopeQuery query.Query
	if len(indexTerms) == 1 {
		scopeQuery = indexTerms[0]
	} else {
		scopeQuery = query.NewDisjunctionQuery(indexTerms...)
	}

	if base == nil {
		return scopeQuery, nil
	}
	return query.NewConjunctionQuery(scopeQuery, base), nil
}
