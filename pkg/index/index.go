// Package index 提供了索引的核心实现。
//
// RAMIndex 是基于内存的索引实现，适合中小规模数据集。
// 它封装了倒排索引、文档存储和并发控制。
package index

import (
	"errors"
	"math"
	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/search"
	"sort"
	"sync"
)

var (
	// ErrClosed 索引已关闭。
	ErrClosed = errors.New("index closed")
	// ErrDocNotFound 文档未找到。
	ErrDocNotFound = errors.New("document not found")
)

// Index 定义了索引的接口。
type Index interface {
	// Add 添加文档到索引。
	Add(doc *document.Document) (int64, error)

	// Delete 从索引中删除文档。
	Delete(docID int64) error

	// Update 更新文档（先删除后添加）。
	Update(docID int64, doc *document.Document) error

	// Search 搜索查询。
	Search(query Query, n int) ([]ScoreDoc, error)

	// DocCount 返回文档数量。
	DocCount() int64

	// GetDocument 获取文档。
	GetDocument(docID int64) (*document.Document, error)

	// Close 关闭索引。
	Close() error
}

// RAMIndex 是基于内存的索引实现。
type RAMIndex struct {
	mu         sync.RWMutex
	inverted   *InvertedIndex // 倒排索引
	docStore   map[int64]*document.Document
	analyzer   analyzer.Analyzer // 分析器
	similarity search.Similarity // 评分算法
	closed     bool
}

// NewRAMIndex 创建新的 RAMIndex。
func NewRAMIndex(analyzerInstance analyzer.Analyzer) *RAMIndex {
	return &RAMIndex{
		inverted:   NewInvertedIndex(),
		docStore:   make(map[int64]*document.Document),
		analyzer:   analyzerInstance,
		similarity: search.NewBM25Similarity(),
	}
}

// NewRAMIndexWithConfig 创建带有配置的 RAMIndex。
func NewRAMIndexWithConfig(opts ...RAMIndexOption) *RAMIndex {
	idx := &RAMIndex{
		inverted:   NewInvertedIndex(),
		docStore:   make(map[int64]*document.Document),
		analyzer:   analyzer.NewStandardAnalyzer(),
		similarity: search.NewBM25Similarity(),
	}
	for _, opt := range opts {
		opt(idx)
	}
	return idx
}

// RAMIndexOption 配置 RAMIndex。
type RAMIndexOption func(*RAMIndex)

// WithAnalyzer 设置分析器。
func WithAnalyzer(a analyzer.Analyzer) RAMIndexOption {
	return func(idx *RAMIndex) {
		idx.analyzer = a
	}
}

// WithSimilarity 设置评分算法。
func WithSimilarity(s search.Similarity) RAMIndexOption {
	return func(idx *RAMIndex) {
		idx.similarity = s
	}
}

// Add 实现了 Index 接口。
func (idx *RAMIndex) Add(doc *document.Document) (int64, error) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return 0, ErrClosed
	}

	docID, err := idx.inverted.AddDocument(doc)
	if err != nil {
		return 0, err
	}
	idx.docStore[docID] = doc.Clone()
	return docID, nil
}

// Delete 实现了 Index 接口。
func (idx *RAMIndex) Delete(docID int64) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return ErrClosed
	}

	if err := idx.inverted.Delete(docID); err != nil {
		return err
	}
	delete(idx.docStore, docID)
	return nil
}

// Update 实现了 Index 接口。
func (idx *RAMIndex) Update(docID int64, doc *document.Document) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return ErrClosed
	}

	if err := idx.inverted.Delete(docID); err != nil {
		return err
	}
	delete(idx.docStore, docID)

	newDocID, err := idx.inverted.AddDocument(doc)
	if err != nil {
		return err
	}
	idx.docStore[newDocID] = doc.Clone()
	return err
}

// Search 实现了 Index 接口。
func (idx *RAMIndex) Search(query Query, n int) ([]ScoreDoc, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.closed {
		return nil, ErrClosed
	}

	if n <= 0 {
		return []ScoreDoc{}, nil
	}

	// 可选快路径：查询对象支持 Top-K 有界收集时，直接使用。
	if qTopN, ok := query.(interface {
		SearchTopN(idx *RAMIndex, n int) ([]ScoreDoc, error)
	}); ok {
		return qTopN.SearchTopN(idx, n)
	}

	results, err := query.Search(idx)
	if err != nil {
		return nil, err
	}
	sort.Sort(ScoreDocs(results))
	return trimTopN(results, n), nil
}

// DocCount 实现了 Index 接口。
func (idx *RAMIndex) DocCount() int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.inverted.DocCount()
}

// GetDocument 获取文档。
func (idx *RAMIndex) GetDocument(docID int64) (*document.Document, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	doc, ok := idx.docStore[docID]
	if !ok {
		return nil, ErrDocNotFound
	}
	return doc.Clone(), nil
}

// Close 实现了 Index 接口。
func (idx *RAMIndex) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.closed {
		return nil
	}
	idx.closed = true
	idx.docStore = make(map[int64]*document.Document)
	return nil
}

// Inverted 返回内部的倒排索引。
func (idx *RAMIndex) Inverted() *InvertedIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.inverted
}

// Analyzer 返回分析器。
func (idx *RAMIndex) Analyzer() analyzer.Analyzer {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.analyzer
}

// Similarity 返回评分算法。
func (idx *RAMIndex) Similarity() search.Similarity {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.similarity
}

// SetSimilarity 设置评分算法。
func (idx *RAMIndex) SetSimilarity(s search.Similarity) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.similarity = s
}

// DocumentIDs 返回当前索引中所有文档 ID 的快照。
func (idx *RAMIndex) DocumentIDs() []int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids := make([]int64, 0, len(idx.docStore))
	for docID := range idx.docStore {
		ids = append(ids, docID)
	}
	return ids
}

// ScoreDoc 表示搜索结果中的一个文档。
type ScoreDoc struct {
	DocID int64   // 文档 ID
	Score float32 // 相关性评分
}

// ScoreDocs 是 ScoreDoc 的切片，实现了排序接口。
type ScoreDocs []ScoreDoc

// Len 实现了 sort.Interface。
func (s ScoreDocs) Len() int {
	return len(s)
}

// Less 实现了 sort.Interface（按评分降序排序）。
func (s ScoreDocs) Less(i, j int) bool {
	if s[i].Score == s[j].Score {
		return s[i].DocID < s[j].DocID
	}
	return s[i].Score > s[j].Score
}

// Swap 实现了 sort.Interface。
func (s ScoreDocs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// TopN 返回评分最高的 n 个结果。
func (s ScoreDocs) TopN(n int) []ScoreDoc {
	if n >= len(s) {
		return s
	}
	result := make([]ScoreDoc, n)
	copy(result, s[:n])
	return result
}

func trimTopN(results []ScoreDoc, n int) []ScoreDoc {
	if n <= 0 {
		return []ScoreDoc{}
	}
	if len(results) <= n {
		return results
	}
	return results[:n]
}

func betterScoreDoc(a, b ScoreDoc) bool {
	if a.Score == b.Score {
		return a.DocID < b.DocID
	}
	return a.Score > b.Score
}

type topKCollector struct {
	n    int
	data []ScoreDoc // 最小堆，堆顶是当前最差结果
}

func newTopKCollector(n int) *topKCollector {
	return &topKCollector{
		n:    n,
		data: make([]ScoreDoc, 0, n),
	}
}

func (c *topKCollector) Add(candidate ScoreDoc) {
	if c.n <= 0 {
		return
	}
	if len(c.data) < c.n {
		c.data = append(c.data, candidate)
		c.siftUp(len(c.data) - 1)
		return
	}
	// 仅当候选比当前最差更好时替换堆顶。
	if betterScoreDoc(candidate, c.data[0]) {
		c.data[0] = candidate
		c.siftDown(0)
	}
}

func (c *topKCollector) Sorted() []ScoreDoc {
	sort.Sort(ScoreDocs(c.data))
	return c.data
}

func (c *topKCollector) Full() bool {
	return c.n > 0 && len(c.data) >= c.n
}

func (c *topKCollector) Worst() ScoreDoc {
	return c.data[0]
}

func (c *topKCollector) siftUp(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if !worseScoreDoc(c.data[i], c.data[p]) {
			break
		}
		c.data[i], c.data[p] = c.data[p], c.data[i]
		i = p
	}
}

func (c *topKCollector) siftDown(i int) {
	n := len(c.data)
	for {
		l := i*2 + 1
		r := l + 1
		smallest := i
		if l < n && worseScoreDoc(c.data[l], c.data[smallest]) {
			smallest = l
		}
		if r < n && worseScoreDoc(c.data[r], c.data[smallest]) {
			smallest = r
		}
		if smallest == i {
			return
		}
		c.data[i], c.data[smallest] = c.data[smallest], c.data[i]
		i = smallest
	}
}

func worseScoreDoc(a, b ScoreDoc) bool {
	if a.Score == b.Score {
		return a.DocID > b.DocID
	}
	return a.Score < b.Score
}

// Query 定义了查询的接口。
type Query interface {
	// Search 执行查询并返回匹配的文档。
	Search(idx *RAMIndex) ([]ScoreDoc, error)
}

// TermQuery 是词项查询。
type TermQuery struct {
	Term  string
	Field string
	Boost float32
}

// NewTermQuery 创建新的 TermQuery。
func NewTermQuery(term string) *TermQuery {
	return &TermQuery{
		Term:  term,
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
func (q *TermQuery) Search(idx *RAMIndex) ([]ScoreDoc, error) {
	return q.searchInternal(idx, 0)
}

// SearchTopN 在收集阶段执行有界 Top-K，避免构建全量结果切片。
func (q *TermQuery) SearchTopN(idx *RAMIndex, n int) ([]ScoreDoc, error) {
	return q.searchInternal(idx, n)
}

func (q *TermQuery) searchInternal(idx *RAMIndex, n int) ([]ScoreDoc, error) {
	postings, err := idx.inverted.GetPostings(q.Term)
	if err != nil {
		return nil, err
	}

	// 获取索引统计信息
	numDocs := idx.inverted.DocCount()
	avgLength := idx.inverted.AvgFieldLength()

	docFreq := len(postings.DocIDs)
	useTopN := n > 0 && docFreq > n
	resultsCap := docFreq
	if useTopN {
		// TopN 路径不构建全量结果，避免高频词查询的超大预分配。
		resultsCap = 0
	}
	results := make([]ScoreDoc, 0, resultsCap)
	var collector *topKCollector
	if useTopN {
		collector = newTopKCollector(n)
	}

	bm25, useBM25 := idx.similarity.(*search.BM25Similarity)
	_, useTFIDF := idx.similarity.(*search.TFIDFSimilarity)
	var bm25IDF, bm25K1, bm25B float32
	var bm25TFNumerator, bm25BaseDenominator, bm25LenFactor float32
	var tfidfIDF float32
	if useBM25 && docFreq > 0 && numDocs > 0 {
		bm25IDF = float32(math.Log(float64(numDocs)/float64(docFreq)+1.0)) + 1.0
		bm25K1 = bm25.K1()
		bm25B = bm25.B()
		if avgLength <= 0 {
			avgLength = 1
		}
		avgLen32 := float32(avgLength)
		bm25TFNumerator = bm25K1 + 1
		bm25BaseDenominator = bm25K1 * (1 - bm25B)
		bm25LenFactor = bm25K1 * bm25B / avgLen32
	}
	if useTFIDF && docFreq > 0 && numDocs > 0 {
		tfidfIDF = float32(math.Log(float64(numDocs)/float64(docFreq))) + 1.0
	}

	for i, docID := range postings.DocIDs {
		termFreq := postings.Freqs[i]
		if useTopN && useBM25 && collector.Full() {
			tf := float32(termFreq)
			// BM25 上界：假设最理想文档长度，若上界都进不了 Top-K，则跳过精确打分。
			upperBM25TF := (tf * bm25TFNumerator) / (tf + bm25BaseDenominator)
			upperBound := ScoreDoc{
				DocID: docID,
				Score: bm25IDF * upperBM25TF * q.Boost,
			}
			if !betterScoreDoc(upperBound, collector.Worst()) {
				continue
			}
		}

		docLength := idx.inverted.FieldLength(docID)

		var score float32
		switch {
		case useBM25:
			tf := float32(termFreq)
			bm25TF := (tf * bm25TFNumerator) / (tf + bm25BaseDenominator + bm25LenFactor*float32(docLength))
			score = bm25IDF * bm25TF * q.Boost
		case useTFIDF:
			score = float32(math.Sqrt(float64(termFreq))) * tfidfIDF * q.Boost
		default:
			score = idx.similarity.Score(
				termFreq,
				docFreq,
				docLength,
				avgLength,
				numDocs,
			) * q.Boost
		}

		candidate := ScoreDoc{
			DocID: docID,
			Score: score,
		}
		if useTopN {
			collector.Add(candidate)
			continue
		}
		results = append(results, candidate)
	}

	if useTopN {
		results = collector.Sorted()
		return results, nil
	}

	sort.Sort(ScoreDocs(results))
	return trimTopN(results, n), nil
}
