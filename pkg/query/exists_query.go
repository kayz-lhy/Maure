package query

import (
	"fmt"

	"maure/pkg/index"
)

// ExistsQuery 匹配字段存在的文档。
type ExistsQuery struct {
	Field string
	Boost float32
}

// NewExistsQuery 创建字段存在查询。
func NewExistsQuery(field string) *ExistsQuery {
	return &ExistsQuery{
		Field: field,
		Boost: 1.0,
	}
}

// Search 实现 Query。
func (q *ExistsQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	ids := idx.DocumentIDs()
	results := make([]index.ScoreDoc, 0, len(ids))
	for _, docID := range ids {
		doc, err := idx.GetDocument(docID)
		if err != nil || doc == nil {
			continue
		}
		if len(doc.GetAll(q.Field)) == 0 {
			continue
		}
		results = append(results, index.ScoreDoc{
			DocID: docID,
			Score: q.Boost,
		})
	}
	sortResults(results)
	return results, nil
}

// Explain 实现 Query。
func (q *ExistsQuery) Explain(idx *index.RAMIndex) string {
	_ = idx
	return fmt.Sprintf("ExistsQuery(field=%s)", q.Field)
}
