// Package analyzer 提供了文本分析和分词功能。
//
// 分词是搜索引擎的核心环节，负责将原始文本转换为词项列表。
// Package 提供了可扩展的分析器接口，支持自定义分词器和过滤器。
package analyzer

// Token 代表一个分词结果。
//
// Token 是文本分词后的最小单位，包含词项文本、位置信息和其他属性。
type Token struct {
	Text      string     // 词项文本
	Start     int        // 在原文中的起始位置
	End       int        // 在原文中的结束位置
	Position  int        // 词项在词流中的位置
	FieldName string     // 所属字段名称
	Type      TokenType  // 词项类型
	Flags     TokenFlags // 附加标志
}

// TokenType 表示词项类型。
type TokenType int

const (
	// TokenTypeWord 普通单词。
	TokenTypeWord TokenType = iota
	// TokenTypeNumber 数字。
	TokenTypeNumber
	// TokenTypeEmail 电子邮件地址。
	TokenTypeEmail
	// TokenTypeURL URL。
	TokenTypeURL
	// TokenTypePunctuation 标点符号。
	TokenTypePunctuation
	// TokenTypeStopWord 停用词。
	TokenTypeStopWord
)

// String 返回 TokenType 的字符串表示。
func (t TokenType) String() string {
	switch t {
	case TokenTypeWord:
		return "word"
	case TokenTypeNumber:
		return "number"
	case TokenTypeEmail:
		return "email"
	case TokenTypeURL:
		return "url"
	case TokenTypePunctuation:
		return "punctuation"
	case TokenTypeStopWord:
		return "stopword"
	default:
		return "unknown"
	}
}

// TokenFlags 表示词项的附加标志。
type TokenFlags uint8

const (
	// TokenFlagHasPayload 词项包含 payload。
	TokenFlagHasPayload TokenFlags = 1 << iota
	// TokenFlagIsSynonym 词项是同义词。
	TokenFlagIsSynonym
)

// TokenStream 提供分词结果的迭代访问。
//
// TokenStream 是只读的单向迭代器，用于遍历分词结果。
// 实现了 io.WriterTo 接口，支持直接写入到字节缓冲区。
type TokenStream interface {
	// Next 移动到下一个词项。
	// 如果没有更多词项返回 false。
	Next() bool
	// Current 返回当前词项。
	Current() *Token
	// Reset 重置迭代器到开头。
	Reset()
	// Close 释放资源。
	Close()
}

// TokenFilter 在 TokenStream 基础上提供过滤和转换功能。
//
// TokenFilter 实现了 TokenStream 接口，可以链式组合。
// 每个 Filter 接收上游的 Token 流，转换后输出新的 Token 流。
type TokenFilter interface {
	TokenStream
	// SetSource 设置上游 TokenStream。
	SetSource(source TokenStream)
}

// baseTokenStream 是 TokenStream 的基础实现。
type baseTokenStream struct {
	tokens []*Token
	pos    int // 当前位置，-1 表示未开始
}

// newBaseTokenStream 从 Token 切片创建 TokenStream。
func newBaseTokenStream(tokens []*Token) *baseTokenStream {
	return &baseTokenStream{
		tokens: tokens,
		pos:    -1,
	}
}

// Next 实现了 TokenStream 接口。
func (ts *baseTokenStream) Next() bool {
	if ts.pos+1 < len(ts.tokens) {
		ts.pos++
		return true
	}
	return false
}

// Current 实现了 TokenStream 接口。
func (ts *baseTokenStream) Current() *Token {
	if ts.pos < len(ts.tokens) {
		return ts.tokens[ts.pos]
	}
	return nil
}

// Reset 实现了 TokenStream 接口。
func (ts *baseTokenStream) Reset() {
	ts.pos = 0
}

// Close 实现了 TokenStream 接口。
func (ts *baseTokenStream) Close() {
	ts.tokens = nil
	ts.pos = 0
}
