// Package index 提供了倒排索引的核心实现。
//
// 倒排索引是搜索引擎的核心数据结构，负责：
//   - 存储词项到文档的映射
//   - 支持高效的词项查询
//   - 记录词项在文档中的位置信息
package index

import (
	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/store"
	"sync"
)

// InvertedIndex 是倒排索引的核心数据结构。
//
// InvertedIndex 使用以下结构存储索引数据：
//   - terms: 词项到倒排表的映射
//   - docCount: 已索引的文档数量
//   - nextDocID: 下一个分配的文档 ID
//
// 倒排表（Postings）包含：
//   - DocIDs: 包含该词项的文档 ID 列表
//   - Freqs: 每个文档中词项的出现频率
//   - Positions: 每个文档中词项的位置列表
type InvertedIndex struct {
	mu           sync.RWMutex
	terms        map[string]*store.Postings
	docCount     int64
	nextDocID    int64
	fieldLength  map[int64]int      // 文档字段长度
	analyzer     analyzer.Analyzer   // 分析器实例（复用）
}

// NewInvertedIndex 创建新的倒排索引。
func NewInvertedIndex() *InvertedIndex {
	idx := &InvertedIndex{
		terms:       make(map[string]*store.Postings),
		docCount:    0,
		nextDocID:   1,
		fieldLength: make(map[int64]int),
		analyzer:    analyzer.NewStandardAnalyzer(),
	}
	return idx
}

// AddDocument 添加文档到索引。
//
// 流程：
//   1. 分析文档的所有字段
//   2. 对每个分词后的词项，更新倒排表
//   3. 记录词项在文档中的位置
func (idx *InvertedIndex) AddDocument(doc *document.Document) (int64, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docID := idx.nextDocID
	idx.nextDocID++

	// 记录每个字段的长度
	totalLength := 0

	// 处理每个字段
	for _, field := range doc.Fields {
		if !field.Indexed {
			continue
		}

		var tokens []*analyzer.Token
		if field.Tokenized {
			// 使用复用分析器实例
			stream := idx.analyzer.Analyze(field.Name, field.StringValue())
			for stream.Next() {
				tokens = append(tokens, stream.Current())
			}
			stream.Close()
		} else {
			// 不分词的字段，整个值作为一个词项
			tokens = []*analyzer.Token{{
				Text:     field.StringValue(),
				Position: 0,
				Type:     analyzer.TokenTypeWord,
			}}
		}

		// 更新倒排表
		for i, token := range tokens {
			term := token.Text

			p, ok := idx.terms[term]
			if !ok {
				p = store.NewPostings()
				idx.terms[term] = p
			}
			p.DocIDs = append(p.DocIDs, docID)
			p.Freqs = append(p.Freqs, 1)
			p.Positions = append(p.Positions, []int{i})
		}

		totalLength += len(tokens)
	}

	idx.fieldLength[docID] = totalLength
	idx.docCount++

	return docID, nil
}

// Delete 从索引中删除文档。
func (idx *InvertedIndex) Delete(docID int64) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, exists := idx.fieldLength[docID]; !exists {
		return ErrDocNotFound
	}

	// 收集要删除的词项
	var emptyTerms []string

	// 从每个词项的倒排表中移除该文档
	for term, p := range idx.terms {
		for i, d := range p.DocIDs {
			if d == docID {
				// 移除该文档
				p.DocIDs = append(p.DocIDs[:i], p.DocIDs[i+1:]...)
				p.Freqs = append(p.Freqs[:i], p.Freqs[i+1:]...)
				p.Positions = append(p.Positions[:i], p.Positions[i+1:]...)

				// 如果倒排表为空，标记为删除
				if len(p.DocIDs) == 0 {
					emptyTerms = append(emptyTerms, term)
				}
				break
			}
		}
	}

	// 删除空的词项
	for _, term := range emptyTerms {
		delete(idx.terms, term)
	}

	delete(idx.fieldLength, docID)
	idx.docCount--
	return nil
}

// GetPostings 获取词项的倒排表。
func (idx *InvertedIndex) GetPostings(term string) (*store.Postings, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	p, ok := idx.terms[term]
	if !ok {
		return nil, ErrDocNotFound
	}
	return p, nil
}

// GetTerms 获取所有词项。
func (idx *InvertedIndex) GetTerms() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	terms := make([]string, 0, len(idx.terms))
	for term := range idx.terms {
		terms = append(terms, term)
	}
	return terms
}

// DocCount 返回索引中的文档数量。
func (idx *InvertedIndex) DocCount() int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docCount
}

// FieldLength 返回指定文档的字段总长度。
func (idx *InvertedIndex) FieldLength(docID int64) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.fieldLength[docID]
}

// AvgFieldLength 返回平均字段长度。
func (idx *InvertedIndex) AvgFieldLength() float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if idx.docCount == 0 {
		return 0
	}
	total := 0
	for _, l := range idx.fieldLength {
		total += l
	}
	return float64(total) / float64(idx.docCount)
}

// NumTerms 返回词项总数。
func (idx *InvertedIndex) NumTerms() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.terms)
}

// ContainsTerm 检查词项是否存在。
func (idx *InvertedIndex) ContainsTerm(term string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, ok := idx.terms[term]
	return ok
}
