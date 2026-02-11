package components

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RangeComponent 解析 field:[a TO b] / field:{a TO b}。
type RangeComponent struct{}

// TryParse 实现 FieldExprComponent。
func (RangeComponent) TryParse(field string, expr string, token string) (FieldParseResult, bool, error) {
	if !strings.HasPrefix(expr, "[") && !strings.HasPrefix(expr, "{") {
		return FieldParseResult{}, false, nil
	}

	inclusive := strings.HasPrefix(expr, "[")
	endBracket := "]"
	if !inclusive {
		endBracket = "}"
	}
	if !strings.HasSuffix(expr, endBracket) {
		return FieldParseResult{}, false, fmt.Errorf("invalid range syntax: %s", token)
	}
	content := expr[1 : len(expr)-1]
	lower, upper, ok := splitRangeBounds(content)
	if !ok {
		return FieldParseResult{}, false, fmt.Errorf("range query must contain TO: %s", token)
	}
	if lower == "" || upper == "" {
		return FieldParseResult{}, false, fmt.Errorf("invalid range bounds: %s", token)
	}
	kind, err := inferRangeKind(lower, upper)
	if err != nil {
		return FieldParseResult{}, false, fmt.Errorf("invalid range query %s: %w", token, err)
	}
	return FieldParseResult{Kind: ExprRange, Field: field, Lower: lower, Upper: upper, ValueKind: kind, Inclusive: inclusive}, true, nil
}

func splitRangeBounds(content string) (string, string, bool) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return "", "", false
	}
	toIdx := -1
	for i, f := range fields {
		if strings.EqualFold(f, "TO") {
			toIdx = i
			break
		}
	}
	if toIdx <= 0 || toIdx >= len(fields)-1 {
		return "", "", false
	}
	lower := strings.Join(fields[:toIdx], " ")
	upper := strings.Join(fields[toIdx+1:], " ")
	return lower, upper, true
}

func inferRangeKind(lower string, upper string) (string, error) {
	if _, err := strconv.ParseFloat(lower, 64); err == nil {
		if _, err := strconv.ParseFloat(upper, 64); err == nil {
			return "number", nil
		}
	}

	if _, err := parseRangeTime(lower); err == nil {
		if _, err := parseRangeTime(upper); err == nil {
			return "time", nil
		}
	}

	return "", fmt.Errorf("range supports only numeric/time bounds")
}

func parseRangeTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}
