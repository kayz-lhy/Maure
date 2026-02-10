package query

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"maure/pkg/analyzer"
	"maure/pkg/document"
	"maure/pkg/index"
	"maure/pkg/store"
)

// RangeValueKind 描述范围查询值类型。
type RangeValueKind int

const (
	RangeValueNumber RangeValueKind = iota
	RangeValueTime
)

// RangeQuery 是字段范围查询（首版支持数值和时间）。
type RangeQuery struct {
	Field      string
	Lower      string
	Upper      string
	Kind       RangeValueKind
	Inclusive  bool
	Boost      float32
	lowerNum   float64
	upperNum   float64
	lowerTime  time.Time
	upperTime  time.Time
	parsedOnce bool
}

// NewRangeQuery 创建范围查询。
func NewRangeQuery(field, lower, upper string, kind RangeValueKind, inclusive bool) *RangeQuery {
	return &RangeQuery{
		Field:     strings.TrimSpace(field),
		Lower:     strings.TrimSpace(lower),
		Upper:     strings.TrimSpace(upper),
		Kind:      kind,
		Inclusive: inclusive,
		Boost:     1.0,
	}
}

// WithBoost 设置权重。
func (q *RangeQuery) WithBoost(boost float32) *RangeQuery {
	q.Boost = boost
	return q
}

func (q *RangeQuery) ensureParsed() error {
	if q.parsedOnce {
		return nil
	}

	switch q.Kind {
	case RangeValueNumber:
		lower, err := strconv.ParseFloat(q.Lower, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric lower bound %q", q.Lower)
		}
		upper, err := strconv.ParseFloat(q.Upper, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric upper bound %q", q.Upper)
		}
		q.lowerNum = lower
		q.upperNum = upper
	case RangeValueTime:
		lower, err := parseRangeTime(q.Lower)
		if err != nil {
			return fmt.Errorf("invalid time lower bound %q", q.Lower)
		}
		upper, err := parseRangeTime(q.Upper)
		if err != nil {
			return fmt.Errorf("invalid time upper bound %q", q.Upper)
		}
		q.lowerTime = lower
		q.upperTime = upper
	default:
		return fmt.Errorf("unsupported range kind")
	}

	q.parsedOnce = true
	return nil
}

// Search 实现 Query。
func (q *RangeQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if err := q.ensureParsed(); err != nil {
		return nil, err
	}

	ids := idx.DocumentIDs()
	results := make([]index.ScoreDoc, 0)
	for _, docID := range ids {
		doc, err := idx.GetDocument(docID)
		if err != nil || doc == nil {
			continue
		}
		if !q.docMatches(doc) {
			continue
		}
		results = append(results, index.ScoreDoc{DocID: docID, Score: q.Boost})
	}

	sortResults(results)
	return results, nil
}

func (q *RangeQuery) docMatches(doc *document.Document) bool {
	for _, field := range doc.GetAll(q.Field) {
		switch q.Kind {
		case RangeValueNumber:
			value, ok := fieldAsFloat(field)
			if ok && q.compareNumber(value) {
				return true
			}
		case RangeValueTime:
			value, ok := fieldAsTime(field)
			if ok && q.compareTime(value) {
				return true
			}
		}
	}
	return false
}

func (q *RangeQuery) compareNumber(v float64) bool {
	if q.Inclusive {
		return v >= q.lowerNum && v <= q.upperNum
	}
	return v > q.lowerNum && v < q.upperNum
}

func (q *RangeQuery) compareTime(v time.Time) bool {
	if q.Inclusive {
		return (v.Equal(q.lowerTime) || v.After(q.lowerTime)) && (v.Equal(q.upperTime) || v.Before(q.upperTime))
	}
	return v.After(q.lowerTime) && v.Before(q.upperTime)
}

// Explain 实现 Query。
func (q *RangeQuery) Explain(idx *index.RAMIndex) string {
	_ = idx
	return fmt.Sprintf("RangeQuery(field=%s, lower=%s, upper=%s)", q.Field, q.Lower, q.Upper)
}

// WildcardQuery 是字段前缀通配查询（prefix-only）。
type WildcardQuery struct {
	Field  string
	Prefix string
	Boost  float32
}

// NewWildcardQuery 创建前缀通配查询。
func NewWildcardQuery(field, prefix string) *WildcardQuery {
	return &WildcardQuery{
		Field:  strings.TrimSpace(field),
		Prefix: strings.ToLower(strings.TrimSpace(prefix)),
		Boost:  1.0,
	}
}

// WithBoost 设置权重。
func (q *WildcardQuery) WithBoost(boost float32) *WildcardQuery {
	q.Boost = boost
	return q
}

// Search 实现 Query。
func (q *WildcardQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	terms := idx.Inverted().GetTerms()
	if len(terms) == 0 {
		return nil, nil
	}

	an := idx.Analyzer()
	scoreByDoc := make(map[int64]float32)
	for _, term := range terms {
		if !strings.HasPrefix(term, q.Prefix) {
			continue
		}
		postings, err := idx.Inverted().GetPostings(term)
		if err != nil {
			continue
		}
		mergePostingScores(idx, postings, q.Boost, scoreByDoc)
	}

	results := filterScoresByFieldTermPredicate(idx, scoreByDoc, q.Field, an, func(t string) bool {
		return strings.HasPrefix(t, q.Prefix)
	})
	sortResults(results)
	return results, nil
}

// Explain 实现 Query。
func (q *WildcardQuery) Explain(idx *index.RAMIndex) string {
	_ = idx
	return fmt.Sprintf("WildcardQuery(field=%s, prefix=%s*)", q.Field, q.Prefix)
}

// FuzzyQuery 是字段模糊查询（仅支持编辑距离 1）。
type FuzzyQuery struct {
	Field    string
	Term     string
	Distance int
	Boost    float32
}

// NewFuzzyQuery 创建模糊查询。
func NewFuzzyQuery(field, term string, distance int) *FuzzyQuery {
	return &FuzzyQuery{
		Field:    strings.TrimSpace(field),
		Term:     strings.ToLower(strings.TrimSpace(term)),
		Distance: distance,
		Boost:    1.0,
	}
}

// WithBoost 设置权重。
func (q *FuzzyQuery) WithBoost(boost float32) *FuzzyQuery {
	q.Boost = boost
	return q
}

// Search 实现 Query。
func (q *FuzzyQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	if q.Distance != 1 {
		return nil, fmt.Errorf("unsupported fuzzy distance: %d", q.Distance)
	}
	terms := idx.Inverted().GetTerms()
	an := idx.Analyzer()
	scoreByDoc := make(map[int64]float32)

	for _, term := range terms {
		if editDistanceAtMostOne(term, q.Term) > q.Distance {
			continue
		}
		postings, err := idx.Inverted().GetPostings(term)
		if err != nil {
			continue
		}
		mergePostingScores(idx, postings, q.Boost, scoreByDoc)
	}

	results := filterScoresByFieldTermPredicate(idx, scoreByDoc, q.Field, an, func(t string) bool {
		return editDistanceAtMostOne(strings.ToLower(t), q.Term) <= q.Distance
	})
	sortResults(results)
	return results, nil
}

// Explain 实现 Query。
func (q *FuzzyQuery) Explain(idx *index.RAMIndex) string {
	_ = idx
	return fmt.Sprintf("FuzzyQuery(field=%s, term=%s~%d)", q.Field, q.Term, q.Distance)
}

func mergePostingScores(idx *index.RAMIndex, postings *store.Postings, boost float32, scoreByDoc map[int64]float32) {
	numDocs := idx.Inverted().DocCount()
	avgLength := idx.Inverted().AvgFieldLength()
	similarity := idx.Similarity()
	docFreq := len(postings.DocIDs)

	for i, docID := range postings.DocIDs {
		termFreq := postings.Freqs[i]
		docLength := idx.Inverted().FieldLength(docID)
		score := similarity.Score(termFreq, docFreq, docLength, avgLength, numDocs) * boost
		if score > scoreByDoc[docID] {
			scoreByDoc[docID] = score
		}
	}
}

func filterScoresByFieldTermPredicate(idx *index.RAMIndex, scoreByDoc map[int64]float32, field string, a analyzer.Analyzer, predicate func(string) bool) []index.ScoreDoc {
	results := make([]index.ScoreDoc, 0, len(scoreByDoc))
	for docID, score := range scoreByDoc {
		doc, err := idx.GetDocument(docID)
		if err != nil || doc == nil {
			continue
		}
		if field != "" && !docMatchesFieldTermPredicate(doc, field, a, predicate) {
			continue
		}
		results = append(results, index.ScoreDoc{DocID: docID, Score: score})
	}
	return results
}

func docMatchesFieldTermPredicate(doc *document.Document, fieldName string, a analyzer.Analyzer, predicate func(string) bool) bool {
	for _, field := range doc.GetAll(fieldName) {
		if field.Tokenized {
			stream := a.Analyze(field.Name, field.StringValue())
			for stream.Next() {
				tok := stream.Current()
				if predicate(strings.ToLower(tok.Text)) {
					stream.Close()
					return true
				}
			}
			stream.Close()
			continue
		}

		if predicate(strings.ToLower(field.StringValue())) {
			return true
		}
	}
	return false
}

func fieldAsFloat(field *document.Field) (float64, bool) {
	switch v := field.Value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func fieldAsTime(field *document.Field) (time.Time, bool) {
	switch v := field.Value.(type) {
	case time.Time:
		return v, true
	case string:
		t, err := parseRangeTime(v)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	default:
		return time.Time{}, false
	}
}

func parseRangeTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time: %s", value)
}

func editDistanceAtMostOne(a, b string) int {
	if a == b {
		return 0
	}
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)
	if int(math.Abs(float64(la-lb))) > 1 {
		return 2
	}

	if la == lb {
		diff := 0
		for i := 0; i < la; i++ {
			if ra[i] != rb[i] {
				diff++
				if diff > 1 {
					return 2
				}
			}
		}
		return diff
	}

	if la > lb {
		ra, rb = rb, ra
		la, lb = lb, la
	}

	i, j := 0, 0
	edits := 0
	for i < la && j < lb {
		if ra[i] == rb[j] {
			i++
			j++
			continue
		}
		edits++
		if edits > 1 {
			return 2
		}
		j++
	}
	if j < lb {
		edits++
	}
	if edits > 1 {
		return 2
	}
	return edits
}

func buildScoreDocs(scoreByDoc map[int64]float32) []index.ScoreDoc {
	results := make([]index.ScoreDoc, 0, len(scoreByDoc))
	for docID, score := range scoreByDoc {
		results = append(results, index.ScoreDoc{DocID: docID, Score: score})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}
