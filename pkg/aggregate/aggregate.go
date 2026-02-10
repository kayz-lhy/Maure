package aggregate

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"maure/pkg/document"
)

const (
	AggCount = "count"
)

// Bucket 是分组聚合桶。
type Bucket struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// Result 是聚合结果。
type Result struct {
	Count   int      `json:"count,omitempty"`
	Buckets []Bucket `json:"buckets,omitempty"`
}

// Build 计算 count/group 聚合。
func Build(docs []*document.Document, agg string, group string) (*Result, error) {
	result := &Result{}

	if agg != "" && !strings.EqualFold(strings.TrimSpace(agg), AggCount) {
		return nil, fmt.Errorf("unsupported agg: %s", agg)
	}

	if strings.EqualFold(strings.TrimSpace(agg), AggCount) {
		result.Count = len(docs)
	}

	group = strings.TrimSpace(group)
	if group == "" {
		return result, nil
	}

	spec, err := parseGroupSpec(group)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		key := "(unknown)"
		switch spec.kind {
		case "field":
			if field := doc.Get(spec.value); field != nil {
				val := strings.TrimSpace(field.StringValue())
				if val != "" {
					key = val
				}
			}
		case "time":
			if field := doc.Get("timestamp"); field != nil {
				if t, ok := parseTimestamp(field.StringValue()); ok {
					bucketStart := t.Truncate(spec.window)
					key = bucketStart.Format(time.RFC3339)
				}
			}
		default:
			return nil, fmt.Errorf("unsupported group kind: %s", spec.kind)
		}
		counts[key]++
	}

	result.Buckets = make([]Bucket, 0, len(counts))
	for key, count := range counts {
		result.Buckets = append(result.Buckets, Bucket{
			Key:   key,
			Count: count,
		})
	}
	sort.Slice(result.Buckets, func(i, j int) bool {
		if result.Buckets[i].Count == result.Buckets[j].Count {
			return result.Buckets[i].Key < result.Buckets[j].Key
		}
		return result.Buckets[i].Count > result.Buckets[j].Count
	})

	return result, nil
}

type groupSpec struct {
	kind   string
	value  string
	window time.Duration
}

func parseGroupSpec(group string) (*groupSpec, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		return nil, fmt.Errorf("group is empty")
	}

	if strings.HasPrefix(strings.ToLower(group), "time(") && strings.HasSuffix(group, ")") {
		windowText := strings.TrimSpace(group[len("time(") : len(group)-1])
		window, err := time.ParseDuration(windowText)
		if err != nil || window <= 0 {
			return nil, fmt.Errorf("invalid time group window: %s", windowText)
		}
		return &groupSpec{
			kind:   "time",
			window: window,
		}, nil
	}

	return &groupSpec{
		kind:  "field",
		value: group,
	}, nil
}

func parseTimestamp(ts string) (time.Time, bool) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, ts)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
