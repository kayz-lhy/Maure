package command

import (
	"fmt"
	"strings"
	"unicode"

	"maure/pkg/document"
)

func parseFieldsParam(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	fields := make([]string, 0, len(parts))

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("fields 包含空字段")
		}
		if !isValidFieldName(name) {
			return nil, fmt.Errorf("fields 包含非法字段名: %s", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		fields = append(fields, name)
	}

	return fields, nil
}

func isValidFieldName(name string) bool {
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func parseIncludeDoc(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func buildDocView(doc *document.Document, includeDoc bool, fields []string) *SearchDocView {
	if doc == nil {
		return nil
	}
	if !includeDoc && len(fields) == 0 {
		return nil
	}

	view := &SearchDocView{}
	if includeDoc {
		view.Summary = summarizeDoc(doc)
	}

	if len(fields) > 0 {
		view.Fields = make(map[string]interface{}, len(fields))
		for _, fieldName := range fields {
			if field := doc.Get(fieldName); field != nil {
				view.Fields[fieldName] = field.Value
			}
		}
	}

	return view
}
