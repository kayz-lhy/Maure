// Package query 提供了查询类型和查询解析器。
//
// 支持的查询类型：
//   - TermQuery：词项查询
//   - BooleanQuery：布尔查询（AND/OR/NOT）
//
// 查询解析器将用户输入的查询字符串解析为查询对象。
package query

import (
	"maure/pkg/index"
	"sort"
)

// Query 是所有查询类型的接口。
//
// Query 定义了搜索引擎查询的统一接口。
// 每种查询类型都实现这个接口，提供 Search 方法执行查询。
type Query interface {
	// Search 执行查询并返回匹配的文档。
	Search(idx *index.RAMIndex) ([]index.ScoreDoc, error)

	// Explain 返回查询的解释信息（用于调试）。
	Explain(idx *index.RAMIndex) string
}

// BooleanOperator 定义了布尔查询的操作符。
type BooleanOperator int

const (
	// OperatorAnd 表示 AND 操作。
	OperatorAnd BooleanOperator = iota
	// OperatorOr 表示 OR 操作。
	OperatorOr
	// OperatorNot 表示 NOT 操作。
	OperatorNot
)

// String 返回操作符的字符串表示。
func (op BooleanOperator) String() string {
	switch op {
	case OperatorAnd:
		return "AND"
	case OperatorOr:
		return "OR"
	case OperatorNot:
		return "NOT"
	default:
		return "UNKNOWN"
	}
}

// BooleanQuery 是布尔查询的实现。
//
// BooleanQuery 组合多个子查询，支持 AND、OR、NOT 三种操作。
//
// 示例：
//   - AND 查询：+go +programming（同时包含 go 和 programming）
//   - OR 查询：go | python（包含 go 或 python）
//   - NOT 查询：programming -java（包含 programming 但不包含 java）
type BooleanQuery struct {
	clauses []BooleanClause // 查询子句
}

// BooleanClause 表示布尔查询中的一个子句。
type BooleanClause struct {
	query Query   // 子查询
	occur Occur   // 发生条件
	boost float32 // 权重
}

// Occur 定义了子句的发生条件。
type Occur int

const (
	// OccurMust 子句必须匹配（AND）。
	OccurMust Occur = iota
	// OccurShould 子句应该匹配（OR）。
	OccurShould
	// OccurMustNot 子句必须不匹配（NOT）。
	OccurMustNot
)

// String 返回 Occur 的字符串表示。
func (o Occur) String() string {
	switch o {
	case OccurMust:
		return "MUST"
	case OccurShould:
		return "SHOULD"
	case OccurMustNot:
		return "MUST_NOT"
	default:
		return "UNKNOWN"
	}
}

// NewBooleanQuery 创建新的布尔查询。
func NewBooleanQuery() *BooleanQuery {
	return &BooleanQuery{
		clauses: make([]BooleanClause, 0),
	}
}

// Add 添加子句到布尔查询。
func (q *BooleanQuery) Add(subQuery Query, occur Occur, boost float32) {
	q.clauses = append(q.clauses, BooleanClause{
		query: subQuery,
		occur: occur,
		boost: boost,
	})
}

// Search 实现了 Query 接口。
func (q *BooleanQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if len(q.clauses) == 0 {
		return nil, nil
	}

	// 分离 MUST、SHOULD、MUST_NOT 子句
	var mustClauses []BooleanClause
	var shouldClauses []BooleanClause
	var mustNotClauses []BooleanClause

	for _, clause := range q.clauses {
		switch clause.occur {
		case OccurMust:
			mustClauses = append(mustClauses, clause)
		case OccurShould:
			shouldClauses = append(shouldClauses, clause)
		case OccurMustNot:
			mustNotClauses = append(mustNotClauses, clause)
		}
	}

	// 如果有 MUST 子句，结果必须在所有 MUST 子句中
	if len(mustClauses) > 0 {
		// 取所有 MUST 子句的交集
		mustResults := make(map[int64]float32)

		for i, clause := range mustClauses {
			results, err := clause.query.Search(idx)
			if err != nil {
				return nil, err
			}

			// 构建当前子查询的文档集合
			currentDocs := make(map[int64]float32)
			for _, r := range results {
				currentDocs[r.DocID] = r.Score * clause.boost
			}

			if i == 0 {
				// 第一个子句，初始化
				for docID, score := range currentDocs {
					mustResults[docID] = score
				}
			} else {
				// 取交集
				for docID := range mustResults {
					if score, ok := currentDocs[docID]; ok {
						// 取平均分
						mustResults[docID] = (mustResults[docID] + score) / 2
					} else {
						delete(mustResults, docID)
					}
				}
			}

			// 如果交集为空，提前返回
			if len(mustResults) == 0 {
				return []index.ScoreDoc{}, nil
			}
		}

		// 收集 MUST_NOT 子句中匹配的文档
		excludeSet := make(map[int64]bool)
		for _, clause := range mustNotClauses {
			results, _ := clause.query.Search(idx)
			for _, r := range results {
				excludeSet[r.DocID] = true
			}
		}

		// 过滤排除的文档
		var results []index.ScoreDoc
		for docID, score := range mustResults {
			if !excludeSet[docID] {
				results = append(results, index.ScoreDoc{
					DocID: docID,
					Score: score,
				})
			}
		}

		// 按评分排序
		sortResults(results)

		return results, nil
	}

	// 没有 MUST 子句，合并 SHOULD 结果
	shouldResults := make(map[int64]index.ScoreDoc)

	// 合并 SHOULD 结果
	for _, clause := range shouldClauses {
		results, err := clause.query.Search(idx)
		if err != nil {
			continue
		}
		for _, r := range results {
			existing := shouldResults[r.DocID]
			newScore := r.Score * clause.boost
			if newScore > existing.Score {
				shouldResults[r.DocID] = index.ScoreDoc{
					DocID: r.DocID,
					Score: newScore,
				}
			}
		}
	}

	// 收集 MUST_NOT 子句中匹配的文档
	excludeSet := make(map[int64]bool)
	for _, clause := range mustNotClauses {
		results, _ := clause.query.Search(idx)
		for _, r := range results {
			excludeSet[r.DocID] = true
		}
	}

	// 过滤排除的文档
	var results []index.ScoreDoc
	for docID, scoreDoc := range shouldResults {
		if !excludeSet[docID] {
			results = append(results, scoreDoc)
		}
	}

	// 按评分排序
	sortResults(results)

	return results, nil
}

// Explain 实现了 Query 接口。
func (q *BooleanQuery) Explain(idx *index.RAMIndex) string {
	result := "BooleanQuery("
	for i, clause := range q.clauses {
		if i > 0 {
			result += " "
		}
		result += clause.occur.String() + ":"
		result += "[...]"
	}
	result += ")"
	return result
}

// sortResults 按评分降序排序结果（使用快速排序）。
func sortResults(results []index.ScoreDoc) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// DisjunctionQuery 是 OR 查询的实现。
type DisjunctionQuery struct {
	queries []Query
}

// NewDisjunctionQuery 创建新的 OR 查询。
func NewDisjunctionQuery(queries ...Query) *DisjunctionQuery {
	return &DisjunctionQuery{
		queries: queries,
	}
}

// Search 实现了 Query 接口。
func (q *DisjunctionQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if len(q.queries) == 0 {
		return nil, nil
	}

	// 合并所有查询结果
	results := make(map[int64]index.ScoreDoc)
	for _, subQuery := range q.queries {
		subResults, err := subQuery.Search(idx)
		if err != nil {
			return nil, err
		}
		for _, r := range subResults {
			existing := results[r.DocID]
			if r.Score > existing.Score {
				results[r.DocID] = r
			}
		}
	}

	// 转换为切片并排序
	resultSlice := make([]index.ScoreDoc, 0, len(results))
	for _, r := range results {
		resultSlice = append(resultSlice, r)
	}
	sortResults(resultSlice)

	return resultSlice, nil
}

// Explain 实现了 Query 接口。
func (q *DisjunctionQuery) Explain(idx *index.RAMIndex) string {
	return "DisjunctionQuery(OR [...])"
}

// ConjunctionQuery 是 AND 查询的实现。
type ConjunctionQuery struct {
	queries []Query
}

// NewConjunctionQuery 创建新的 AND 查询。
func NewConjunctionQuery(queries ...Query) *ConjunctionQuery {
	return &ConjunctionQuery{
		queries: queries,
	}
}

// Search 实现了 Query 接口。
func (q *ConjunctionQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if len(q.queries) == 0 {
		return nil, nil
	}

	// 取第一个查询的结果
	firstResults, err := q.queries[0].Search(idx)
	if err != nil {
		return nil, err
	}

	// 收集文档 ID
	docIDs := make(map[int64]float32)
	for _, r := range firstResults {
		docIDs[r.DocID] = r.Score
	}

	// 依次与后续查询取交集
	for i := 1; i < len(q.queries); i++ {
		subResults, err := q.queries[i].Search(idx)
		if err != nil {
			return nil, err
		}

		// 构建子查询的文档集合
		subDocIDs := make(map[int64]float32)
		for _, r := range subResults {
			subDocIDs[r.DocID] = r.Score
		}

		// 取交集
		for docID := range docIDs {
			if score, ok := subDocIDs[docID]; ok {
				docIDs[docID] = (docIDs[docID] + score) / 2
			} else {
				delete(docIDs, docID)
			}
		}
	}

	// 转换为切片并排序
	resultSlice := make([]index.ScoreDoc, 0, len(docIDs))
	for docID, score := range docIDs {
		resultSlice = append(resultSlice, index.ScoreDoc{
			DocID: docID,
			Score: score,
		})
	}
	sortResults(resultSlice)

	return resultSlice, nil
}

// Explain 实现了 Query 接口。
func (q *ConjunctionQuery) Explain(idx *index.RAMIndex) string {
	return "ConjunctionQuery(AND [...])"
}
