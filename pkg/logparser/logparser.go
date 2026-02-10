package logparser

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"maure/pkg/document"
)

// LogParser 将单行日志解析为可检索文档。
type LogParser interface {
	Parse(line string) (*document.Document, error)
}

const (
	FormatAuto    = "auto"
	FormatJSON    = "json"
	FormatLogback = "logback"
)

// ParserConstructor 是具体解析器构造函数。
type ParserConstructor func() LogParser

// ParserRegistry 管理解析器注册与实例化。
type ParserRegistry struct {
	mu           sync.RWMutex
	constructors map[string]ParserConstructor
}

// NewParserRegistry 创建空注册中心。
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		constructors: make(map[string]ParserConstructor),
	}
}

// Register 注册具体解析器构造器。
func (r *ParserRegistry) Register(format string, ctor ParserConstructor) error {
	key := normalizeFormat(format)
	if key == "" {
		return fmt.Errorf("parser format is empty")
	}
	if key == FormatAuto {
		return fmt.Errorf("format %q is reserved", FormatAuto)
	}
	if ctor == nil {
		return fmt.Errorf("parser constructor is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[key] = ctor
	return nil
}

// Get 创建指定格式的解析器实例。
func (r *ParserRegistry) Get(format string) (LogParser, error) {
	key := normalizeFormat(format)
	if key == "" {
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	r.mu.RLock()
	ctor, ok := r.constructors[key]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}
	return ctor(), nil
}

// GetForLine 根据指定模式和单行内容选择解析器。
func (r *ParserRegistry) GetForLine(format string, line string) (LogParser, error) {
	key := normalizeFormat(format)
	if key == FormatAuto {
		return r.Get(DetectFormat(line))
	}
	return r.Get(key)
}

// Formats 返回已注册格式列表（排序后）。
func (r *ParserRegistry) Formats() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	formats := make([]string, 0, len(r.constructors))
	for k := range r.constructors {
		formats = append(formats, k)
	}
	sort.Strings(formats)
	return formats
}

var defaultRegistry = newDefaultRegistry()

func newDefaultRegistry() *ParserRegistry {
	reg := NewParserRegistry()
	_ = reg.Register(FormatJSON, func() LogParser { return &JSONParser{} })
	_ = reg.Register(FormatLogback, func() LogParser { return &LogbackParser{} })
	return reg
}

// RegisterParser 在默认注册中心注册解析器（用于扩展）。
func RegisterParser(format string, ctor ParserConstructor) error {
	return defaultRegistry.Register(format, ctor)
}

// GetParser 返回指定格式的解析器（兼容旧 API）。
func GetParser(format string) (LogParser, error) {
	return defaultRegistry.Get(format)
}

// DetectFormat 根据内容做简单格式识别。
func DetectFormat(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return FormatJSON
	}
	return FormatLogback
}

// GetParserForLine 根据配置与内容选择解析器（兼容旧 API）。
func GetParserForLine(format string, line string) (LogParser, error) {
	return defaultRegistry.GetForLine(format, line)
}

func normalizeFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}
