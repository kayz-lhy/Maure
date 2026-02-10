package logparser

import (
	"fmt"
	"regexp"
	"strings"

	"maure/pkg/document"
)

var logbackPattern = regexp.MustCompile(`^\s*(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d{3})?)\s+([A-Za-z]+)\s+(\S+)\s+-\s+(.*)$`)

// LogbackParser 解析常见 Logback 单行格式。
type LogbackParser struct{}

// Parse 支持格式：
// 2026-02-10 12:34:56.789 INFO com.example.App - hello world
func (p *LogbackParser) Parse(line string) (*document.Document, error) {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return nil, fmt.Errorf("empty log line")
	}

	matches := logbackPattern.FindStringSubmatch(raw)
	if len(matches) != 5 {
		return nil, fmt.Errorf("invalid logback line")
	}

	fields := map[string]interface{}{
		"timestamp": matches[1],
		"level":     matches[2],
		"class":     matches[3],
		"message":   matches[4],
	}
	return BuildDocumentFromFields(raw, fields), nil
}
