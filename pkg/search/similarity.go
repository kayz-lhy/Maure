// Package search 提供了相关性评分算法。
//
// 支持多种评分算法：
//   - TF-IDF：经典的词频-逆文档频率算法
//   - BM25：现代的概率评分模型
//
// 评分算法用于衡量查询与文档的相关程度。
package search

import (
	"math"
	"maure/pkg/store"
)

// Similarity 定义了评分算法的接口。
//
// Similarity 负责计算查询词项与文档的相关性评分。
// 不同的评分算法实现会考虑不同的因素（词频、文档长度、IDF 等）。
type Similarity interface {
	// Score 计算单个词项的评分。
	//
	// termFreq: 词项在文档中的出现频率
	// docFreq: 包含该词项的文档数量
	// docLength: 文档中该字段的总词数
	// avgDocLength: 索引中该字段的平均文档长度
	// numDocs: 索引中的文档总数
	Score(termFreq int, docFreq int, docLength int, avgDocLength float64, numDocs int64) float32

	// Name 返回评分算法的名称。
	Name() string
}

// TFIDFSimilarity 实现了 TF-IDF 评分算法。
//
// TF-IDF 是经典的向量空间模型评分算法：
//   TF(t,d) = sqrt(词项 t 在文档 d 中的频率)
//   IDF(t) = log(总文档数 / 包含 t 的文档数)
//   Score = TF(t,d) * IDF(t)
//
// 特点：
//   - 简单直观，易于理解
//   - 对词频敏感
//   - 不考虑文档长度归一化
type TFIDFSimilarity struct{}

// NewTFIDFSimilarity 创建新的 TF-IDF 评分器。
func NewTFIDFSimilarity() *TFIDFSimilarity {
	return &TFIDFSimilarity{}
}

// Score 实现了 Similarity 接口。
func (s *TFIDFSimilarity) Score(termFreq int, docFreq int, docLength int, avgDocLength float64, numDocs int64) float32 {
	if termFreq == 0 || docFreq == 0 || numDocs == 0 {
		return 0
	}

	// TF: sqrt(tf)
	tf := float32(math.Sqrt(float64(termFreq)))

	// IDF: log(N/df) + 1（避免 log(1) = 0）
	idf := float32(math.Log(float64(numDocs)/float64(docFreq))) + 1.0

	return tf * idf
}

// Name 实现了 Similarity 接口。
func (s *TFIDFSimilarity) Name() string {
	return "TF-IDF"
}

// BM25Similarity 实现了 BM25 评分算法。
//
// BM25 是 Okapi BM25，是一种基于概率模型的评分算法：
//   Score = IDF(t) * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl/avgdl))
//
// 参数说明：
//   k1：词频饱和度参数，通常 1.2-2.0
//   b：文档长度归一化参数，通常 0.75
//
// 特点：
//   - 对词频有饱和处理（不会无限增长）
//   - 考虑文档长度归一化
//   - 在实际应用中效果通常优于 TF-IDF
type BM25Similarity struct {
	k1 float32 // 词频饱和参数
	b  float32 // 长度归一化参数
}

// DefaultBM25Params BM25 的默认参数。
const DefaultBM25Params = 1.2

// NewBM25Similarity 创建新的 BM25 评分器。
func NewBM25Similarity() *BM25Similarity {
	return &BM25Similarity{
		k1: DefaultBM25Params,
		b:  0.75,
	}
}

// NewBM25SimilarityWithParams 创建带有自定义参数的 BM25 评分器。
func NewBM25SimilarityWithParams(k1, b float32) *BM25Similarity {
	return &BM25Similarity{k1: k1, b: b}
}

// Score 实现了 Similarity 接口。
func (s *BM25Similarity) Score(termFreq int, docFreq int, docLength int, avgDocLength float64, numDocs int64) float32 {
	if termFreq == 0 || docFreq == 0 || numDocs == 0 {
		return 0
	}

	// IDF: log((N - n + 0.5) / (n + 0.5) + 1)
	// 简化版本：log(N/n + 1)
	n := float64(docFreq)
	N := float64(numDocs)
	idf := float32(math.Log(N/n+1)) + 1.0

	// TF 部分：BM25 公式
	dl := float64(docLength)
	if avgDocLength <= 0 {
		avgDocLength = 1
	}
	lengthNorm := float64(s.b)*dl/avgDocLength + (1 - float64(s.b))

	bm25Tf := (float32(termFreq) * (s.k1 + 1)) / (float32(termFreq) + s.k1*float32(lengthNorm))

	return idf * bm25Tf
}

// Name 实现了 Similarity 接口。
func (s *BM25Similarity) Name() string {
	return "BM25"
}

// K1 返回 BM25 的 k1 参数。
func (s *BM25Similarity) K1() float32 {
	return s.k1
}

// BM25L 是 BM25 的变体，使用不同的长度归一化。
type BM25L struct {
	k1 float32
	b  float32
}

// NewBM25L 创建新的 BM25L 评分器。
func NewBM25L() *BM25L {
	return &BM25L{k1: 1.2, b: 0.75}
}

// Score 实现了 Similarity 接口。
func (s *BM25L) Score(termFreq int, docFreq int, docLength int, avgDocLength float64, numDocs int64) float32 {
	if termFreq == 0 || docFreq == 0 || numDocs == 0 {
		return 0
	}

	// IDF: log(N/n + 1)
	n := float64(docFreq)
	N := float64(numDocs)
	idf := float32(math.Log(N/n+1)) + 1.0

	// TF 部分
	tf := float64(termFreq)
	dl := float64(docLength)
	if avgDocLength <= 0 {
		avgDocLength = 1
	}

	// BM25L 使用不同的归一化公式
	delta := float64(1.0)
	lengthNorm := float64(s.b) * dl / avgDocLength
	bm25LTf := (tf*(float64(s.k1)+1) + delta) / (tf + float64(s.k1)*(float64(1-s.b)+lengthNorm) + delta)

	return idf * float32(bm25LTf)
}

// Name 实现了 Similarity 接口。
func (s *BM25L) Name() string {
	return "BM25L"
}

// CollectionStatistics 收集索引的统计信息。
type CollectionStatistics struct {
	NumDocs      int64            // 文档总数
	AvgDocLength float64          // 平均文档长度
	FieldLengths map[int64]int    // 每个文档的长度
	Postings     map[string]*store.Postings // 词项倒排表
}

// NewCollectionStatistics 创建统计信息。
func NewCollectionStatistics(numDocs int64, avgDocLength float64) *CollectionStatistics {
	return &CollectionStatistics{
		NumDocs:      numDocs,
		AvgDocLength: avgDocLength,
		FieldLengths: make(map[int64]int),
		Postings:     make(map[string]*store.Postings),
	}
}

// Scorer 用于计算文档对查询的评分。
type Scorer struct {
	similarity Similarity
	stats      *CollectionStatistics
}

// NewScorer 创建新的评分器。
func NewScorer(similarity Similarity, stats *CollectionStatistics) *Scorer {
	return &Scorer{
		similarity: similarity,
		stats:      stats,
	}
}

// ScoreTerm 对单个词项计算评分。
func (s *Scorer) ScoreTerm(term string, docID int64) float32 {
	postings, ok := s.stats.Postings[term]
	if !ok {
		return 0
	}

	// 找到词项在文档中的频率
	termFreq := 0
	for i, id := range postings.DocIDs {
		if id == docID {
			termFreq = postings.Freqs[i]
			break
		}
	}

	if termFreq == 0 {
		return 0
	}

	return s.similarity.Score(
		termFreq,
		len(postings.DocIDs),
		s.stats.FieldLengths[docID],
		s.stats.AvgDocLength,
		s.stats.NumDocs,
	)
}

// ScoreTerms 对多个词项计算评分并求和。
func (s *Scorer) ScoreTerms(terms []string, docID int64) float32 {
	var total float32
	for _, term := range terms {
		total += s.ScoreTerm(term, docID)
	}
	return total
}

// DefaultSimilarity 返回默认的评分算法。
func DefaultSimilarity() Similarity {
	return NewBM25Similarity()
}
