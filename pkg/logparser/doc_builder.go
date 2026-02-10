package logparser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"maure/pkg/document"
)

// BuildDocumentFromFields 将解析后的字段映射构造成文档。
func BuildDocumentFromFields(raw string, fields map[string]interface{}) *document.Document {
	doc := document.NewDocument()
	doc.Add(document.NewTextField("raw", raw))

	for k, v := range fields {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if isCommonKey(key) {
			continue
		}
		addParsedField(doc, key, v)
	}

	normalizeCommonFields(doc, fields)
	return doc
}

func addParsedField(doc *document.Document, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		// 默认字符串字段作为可分词文本，保证可搜索。
		doc.Add(document.NewTextField(key, v))
	case json.Number:
		if i, err := v.Int64(); err == nil {
			doc.Add(document.NewInt64Field(key, i))
			return
		}
		if f, err := v.Float64(); err == nil {
			doc.Add(document.NewFloat64Field(key, f))
			return
		}
		doc.Add(document.NewStoredField(key, v.String()))
	case float64:
		doc.Add(document.NewFloat64Field(key, v))
	case int:
		doc.Add(document.NewInt64Field(key, int64(v)))
	case int64:
		doc.Add(document.NewInt64Field(key, v))
	case bool:
		doc.Add(document.NewStoredField(key, v))
	default:
		doc.Add(document.NewStoredField(key, fmt.Sprintf("%v", v)))
	}
}

func normalizeCommonFields(doc *document.Document, fields map[string]interface{}) {
	if msg, ok := pickString(fields, "message", "msg"); ok {
		doc.Add(document.NewTextField("message", msg))
	}
	if lvl, ok := pickString(fields, "level", "severity"); ok {
		doc.Add(document.NewStringField("level", strings.ToUpper(strings.TrimSpace(lvl))))
	}
	if logger, ok := pickString(fields, "logger", "class", "source"); ok {
		doc.Add(document.NewStringField("logger", logger))
	}
	if ts, ok := pickString(fields, "timestamp", "time", "ts"); ok {
		doc.Add(document.NewStringField("timestamp", normalizeTimestamp(ts)))
	}
}

func isCommonKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "message", "msg", "level", "severity", "logger", "class", "source", "timestamp", "time", "ts":
		return true
	default:
		return false
	}
}

func pickString(fields map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		v, ok := fields[key]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case string:
			if strings.TrimSpace(val) != "" {
				return val, true
			}
		case json.Number:
			return val.String(), true
		case float64:
			return strconv.FormatFloat(val, 'f', -1, 64), true
		case int:
			return strconv.Itoa(val), true
		case int64:
			return strconv.FormatInt(val, 10), true
		}
	}
	return "", false
}

func normalizeTimestamp(ts string) string {
	candidates := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"2006-01-02 15:04:05",
	}

	for _, layout := range candidates {
		t, err := time.Parse(layout, ts)
		if err == nil {
			return t.Format(time.RFC3339Nano)
		}
	}
	return ts
}
