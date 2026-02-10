package query

import (
	"fmt"
	"maure/pkg/index"
	"maure/pkg/store"
	"strings"
)

// TermQuery 是词项查询。
//
// TermQuery 是最基本的查询类型，用于精确匹配词项。
type TermQuery struct {
	Term  string
	Field string
	Boost float32
}

// NewTermQuery 创建新的 TermQuery。
func NewTermQuery(term string) *TermQuery {
	return &TermQuery{
		Term:  strings.ToLower(term),
		Boost: 1.0,
	}
}

// WithField 设置查询字段。
func (q *TermQuery) WithField(field string) *TermQuery {
	q.Field = field
	return q
}

// WithBoost 设置权重。
func (q *TermQuery) WithBoost(boost float32) *TermQuery {
	q.Boost = boost
	return q
}

// Search 实现了 Query 接口。
func (q *TermQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	postings, err := idx.Inverted().GetPostings(q.Term)
	if err != nil {
		return nil, err
	}

	// 获取索引统计信息
	numDocs := idx.Inverted().DocCount()
	avgLength := idx.Inverted().AvgFieldLength()
	similarity := idx.Similarity()

	// 计算每个文档的评分
	results := make([]index.ScoreDoc, 0, len(postings.DocIDs))
	for i, docID := range postings.DocIDs {
		termFreq := postings.Freqs[i]
		docLength := idx.Inverted().FieldLength(docID)

		// 使用评分算法计算
		score := similarity.Score(
			termFreq,
			len(postings.DocIDs),
			docLength,
			avgLength,
			numDocs,
		) * q.Boost

		results = append(results, index.ScoreDoc{
			DocID: docID,
			Score: score,
		})
	}

	// 按评分降序排序
	sortResults(results)

	return results, nil
}

// Explain 实现了 Query 接口。
func (q *TermQuery) Explain(idx *index.RAMIndex) string {
	return fmt.Sprintf("TermQuery(term=%s, boost=%.4f)", q.Term, q.Boost)
}

// PhraseQuery 是短语查询的实现。
type PhraseQuery struct {
	terms []string
	slop  int // 词项之间的最大距离
	boost float32
}

// NewPhraseQuery 创建新的短语查询。
func NewPhraseQuery(terms ...string) *PhraseQuery {
	// 自动转小写以匹配索引
	lowerTerms := make([]string, len(terms))
	for i, term := range terms {
		lowerTerms[i] = strings.ToLower(term)
	}
	return &PhraseQuery{
		terms: lowerTerms,
		slop:  0,
		boost: 1.0,
	}
}

// WithSlop 设置词项之间的最大距离。
func (q *PhraseQuery) WithSlop(slop int) *PhraseQuery {
	q.slop = slop
	return q
}

// WithBoost 设置权重。
func (q *PhraseQuery) WithBoost(boost float32) *PhraseQuery {
	q.boost = boost
	return q
}

// Search 实现了 Query 接口。
func (q *PhraseQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if len(q.terms) == 0 {
		return nil, nil
	}

	// 获取第一个词项的倒排表
	firstTerm := q.terms[0]
	postings, err := idx.Inverted().GetPostings(firstTerm)
	if err != nil {
		return nil, err
	}

	numDocs := idx.Inverted().DocCount()
	avgLength := idx.Inverted().AvgFieldLength()
	similarity := idx.Similarity()

	// 对每个候选文档检查短语匹配
	results := make([]index.ScoreDoc, 0)
	for i, docID := range postings.DocIDs {
		// 获取第一个词项的位置
		firstPositions := postings.Positions[i]

		// 检查短语匹配
		if q.matchPhraseInDoc(idx, docID, firstPositions) {
			// 计算评分
			termFreq := postings.Freqs[i]
			docLength := idx.Inverted().FieldLength(docID)

			score := similarity.Score(
				termFreq,
				len(postings.DocIDs),
				docLength,
				avgLength,
				numDocs,
			) * q.boost

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

// matchPhraseInDoc 检查文档中是否匹配短语。
func (q *PhraseQuery) matchPhraseInDoc(idx *index.RAMIndex, docID int64, firstPositions []int) bool {
	if len(q.terms) == 1 {
		return true
	}

	// 获取后续词项的倒排表
	for i := 1; i < len(q.terms); i++ {
		postings, err := idx.Inverted().GetPostings(q.terms[i])
		if err != nil {
			return false
		}

		// 找到该文档中的位置
		positions := q.getPositionsForDoc(postings, docID)
		if len(positions) == 0 {
			return false
		}

		// 检查位置是否满足 slop 要求
		found := false
		for _, firstPos := range firstPositions {
			for _, pos := range positions {
				distance := pos - firstPos - i
				if distance >= 0 && distance <= q.slop {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// getPositionsForDoc 获取指定文档的位置列表。
func (q *PhraseQuery) getPositionsForDoc(postings *store.Postings, docID int64) []int {
	for i, id := range postings.DocIDs {
		if id == docID {
			return postings.Positions[i]
		}
	}
	return nil
}

// Explain 实现了 Query 接口。
func (q *PhraseQuery) Explain(idx *index.RAMIndex) string {
	result := "PhraseQuery("
	for i, term := range q.terms {
		if i > 0 {
			result += " "
		}
		result += term
	}
	result += ")"
	return result
}
