package query

import (
	"maure/pkg/dsl"
	"maure/pkg/index"
)

// QueryParser 将 DSL 查询字符串解析为执行计划。
type QueryParser struct {
	dslParser *dsl.Parser
}

// NewQueryParser 创建新的查询解析器。
func NewQueryParser() *QueryParser {
	return &QueryParser{dslParser: dsl.NewParser()}
}

// ParsePlan 解析 DSL 并返回执行计划（包含版本、作用域、分页与排序元信息）。
func (p *QueryParser) ParsePlan(s string) (*QueryPlan, error) {
	parsed, err := p.dslParser.Parse(s)
	if err != nil {
		return nil, err
	}
	return CompileDSL(parsed)
}

// Parse 解析 DSL 并返回可执行 Query，保留历史调用兼容。
func (p *QueryParser) Parse(s string) (Query, error) {
	plan, err := p.ParsePlan(s)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}
	return plan.Query, nil
}

// notQuery 是 NOT 查询的实现。
type notQuery struct {
	subQuery Query
}

// Search 实现了 Query 接口。
func (q *notQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	// NOT 查询本身不返回结果，只用于过滤
	return nil, nil
}

// Explain 实现了 Query 接口。
func (q *notQuery) Explain(idx *index.RAMIndex) string {
	return "NOT(...)"
}

// MustQuery 是 MUST 查询的实现（类似于 BooleanQuery 中的 MUST）。
type MustQuery struct {
	query Query
}

// NewMustQuery 创建新的 MUST 查询。
func NewMustQuery(query Query) *MustQuery {
	return &MustQuery{query: query}
}

// Search 实现了 Query 接口。
func (q *MustQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	return q.query.Search(idx)
}

// Explain 实现了 Query 接口。
func (q *MustQuery) Explain(idx *index.RAMIndex) string {
	return "MUST(...)"
}

// ShouldQuery 是 SHOULD 查询的实现。
type ShouldQuery struct {
	queries []Query
}

// NewShouldQuery 创建新的 SHOULD 查询。
func NewShouldQuery(queries ...Query) *ShouldQuery {
	return &ShouldQuery{queries: queries}
}

// Search 实现了 Query 接口。
func (q *ShouldQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	results := make(map[int64]index.ScoreDoc)

	for _, subQuery := range q.queries {
		subResults, _ := subQuery.Search(idx)
		for _, r := range subResults {
			existing := results[r.DocID]
			if r.Score > existing.Score {
				results[r.DocID] = r
			}
		}
	}

	resultSlice := make([]index.ScoreDoc, 0, len(results))
	for _, r := range results {
		resultSlice = append(resultSlice, r)
	}

	sortResults(resultSlice)
	return resultSlice, nil
}

// Explain 实现了 Query 接口。
func (q *ShouldQuery) Explain(idx *index.RAMIndex) string {
	return "SHOULD(...)"
}
