package search

import (
	"math"
	"testing"

	"maure/pkg/store"
)

func TestTFIDFSimilarity_Score(t *testing.T) {
	scorer := NewTFIDFSimilarity()

	tests := []struct {
		name     string
		termFreq int
		docFreq  int
		docLen   int
		avgLen   float64
		numDocs  int64
		want     float32
	}{
		{
			name:     "basic term",
			termFreq: 2,
			docFreq:  10,
			docLen:   100,
			avgLen:   100,
			numDocs:  1000,
			want:     float32(math.Sqrt(2)) * (float32(math.Log(1000/10)) + 1),
		},
		{
			name:     "rare term",
			termFreq: 1,
			docFreq:  1,
			docLen:   50,
			avgLen:   100,
			numDocs:  1000,
			want:     float32(math.Sqrt(1)) * (float32(math.Log(1000/1)) + 1),
		},
		{
			name:     "zero freq",
			termFreq: 0,
			docFreq:  10,
			docLen:   100,
			avgLen:   100,
			numDocs:  1000,
			want:     0,
		},
		{
			name:     "zero docFreq",
			termFreq: 2,
			docFreq:  0,
			docLen:   100,
			avgLen:   100,
			numDocs:  1000,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.Score(tt.termFreq, tt.docFreq, tt.docLen, tt.avgLen, tt.numDocs)
			// 由于浮点精度，使用近似比较
			if got < tt.want-0.001 || got > tt.want+0.001 {
				t.Errorf("Score() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBM25Similarity_Score(t *testing.T) {
	scorer := NewBM25Similarity()

	tests := []struct {
		name     string
		termFreq int
		docFreq  int
		docLen   int
		avgLen   float64
		numDocs  int64
	}{
		{
			name:     "basic term",
			termFreq: 2,
			docFreq:  10,
			docLen:   100,
			avgLen:   100,
			numDocs:  1000,
		},
		{
			name:     "short doc",
			termFreq: 3,
			docFreq:  5,
			docLen:   20,
			avgLen:   100,
			numDocs:  500,
		},
		{
			name:     "zero freq",
			termFreq: 0,
			docFreq:  10,
			docLen:   100,
			avgLen:   100,
			numDocs:  1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.Score(tt.termFreq, tt.docFreq, tt.docLen, tt.avgLen, tt.numDocs)

			if tt.termFreq == 0 || tt.docFreq == 0 || tt.numDocs == 0 {
				if got != 0 {
					t.Errorf("Score() = %v, want 0", got)
				}
				return
			}

			// 评分应该为正
			if got <= 0 {
				t.Errorf("Score() = %v, want > 0", got)
			}
		})
	}
}

func TestBM25Similarity_Saturation(t *testing.T) {
	// 测试 BM25 的词频饱和特性
	scorer := NewBM25Similarity()

	docLen := 100.0
	avgLen := 100.0
	docFreq := 10
	numDocs := int64(1000)

	scores := make([]float32, 0)
	for tf := 1; tf <= 10; tf++ {
		score := scorer.Score(tf, docFreq, int(docLen), avgLen, numDocs)
		scores = append(scores, score)
	}

	// 评分应该递增但增速减缓（饱和特性）
	for i := 1; i < len(scores); i++ {
		if scores[i] < scores[i-1] {
			t.Errorf("Score should be monotonically increasing, but %v < %v",
				scores[i], scores[i-1])
		}
	}

	// 相邻评分的增量应该递减
	for i := 2; i < len(scores); i++ {
		delta1 := scores[i-1] - scores[i-2]
		delta2 := scores[i] - scores[i-1]
		if delta2 > delta1 {
			t.Errorf("Delta should decrease (saturation), but %v > %v",
				delta2, delta1)
		}
	}
}

func TestBM25WithParams(t *testing.T) {
	// 测试自定义参数
	k1, b := float32(2.0), float32(0.5)
	scorer := NewBM25SimilarityWithParams(k1, b)

	score := scorer.Score(5, 10, 100, 100, 1000)

	if score <= 0 {
		t.Errorf("Score() = %v, want > 0", score)
	}

	if scorer.Name() != "BM25" {
		t.Errorf("Name() = %v, want BM25", scorer.Name())
	}
}

func TestScorer_ScoreTerm(t *testing.T) {
	sim := NewBM25Similarity()
	stats := NewCollectionStatistics(100, 50.0)
	stats.FieldLengths[1] = 100
	stats.FieldLengths[2] = 50

	// 添加倒排表数据
	stats.Postings["test"] = &store.Postings{
		DocIDs:    []int64{1, 2},
		Freqs:     []int{3, 1},
		Positions: [][]int{{0, 1, 2}, {0}},
	}

	scorer := NewScorer(sim, stats)

	// 测试单个词项评分
	score := scorer.ScoreTerm("test", 1)
	if score <= 0 {
		t.Errorf("ScoreTerm() = %v, want > 0", score)
	}

	// 测试不存在的词项
	score = scorer.ScoreTerm("nonexistent", 1)
	if score != 0 {
		t.Errorf("ScoreTerm() for nonexistent term = %v, want 0", score)
	}
}

func TestScorer_ScoreTerms(t *testing.T) {
	sim := NewBM25Similarity()
	stats := NewCollectionStatistics(100, 50.0)
	stats.FieldLengths[1] = 100

	// 添加倒排表数据
	stats.Postings["test"] = &store.Postings{
		DocIDs:    []int64{1},
		Freqs:     []int{2},
		Positions: [][]int{{0, 1}},
	}
	stats.Postings["search"] = &store.Postings{
		DocIDs:    []int64{1},
		Freqs:     []int{1},
		Positions: [][]int{{2}},
	}

	scorer := NewScorer(sim, stats)

	// 测试多个词项评分
	score := scorer.ScoreTerms([]string{"test", "search"}, 1)
	if score <= 0 {
		t.Errorf("ScoreTerms() = %v, want > 0", score)
	}
}

func TestDefaultSimilarity(t *testing.T) {
	sim := DefaultSimilarity()
	if sim.Name() != "BM25" {
		t.Errorf("DefaultSimilarity() = %v, want BM25", sim.Name())
	}
}

func BenchmarkTFIDFSimilarity(b *testing.B) {
	scorer := NewTFIDFSimilarity()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.Score(5, 100, 100, 50, 10000)
	}
}

func BenchmarkBM25Similarity(b *testing.B) {
	scorer := NewBM25Similarity()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorer.Score(5, 100, 100, 50, 10000)
	}
}
