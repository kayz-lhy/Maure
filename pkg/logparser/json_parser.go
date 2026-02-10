package logparser

import (
	"encoding/json"
	"fmt"
	"strings"

	"maure/pkg/document"
)

// JSONParser 解析单行 JSON 日志。
type JSONParser struct{}

// Parse 解析顶层 JSON 对象并构建文档。
func (p *JSONParser) Parse(line string) (*document.Document, error) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return nil, fmt.Errorf("empty log line")
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()

	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse json log: %w", err)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("json log object is empty")
	}

	return BuildDocumentFromFields(raw, payload), nil
}
