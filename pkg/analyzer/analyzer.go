package analyzer

import (
	"unicode"
)

// Analyzer 定义了文本分析的接口。
//
// Analyzer 负责将原始文本转换为可索引的词项序列。
// 它组合了分词（Tokenization）和可能的过滤处理。
//
// Analyzer 的实现应该是线程安全的。
type Analyzer interface {
	// Analyze 对给定字段和文本进行分析。
	// 返回的 TokenStream 包含所有生成的词项。
	Analyze(fieldName string, text string) TokenStream
	// DefaultField 返回默认分析的字段名称。
	DefaultField() string
}

// Tokenizer 将原始文本分割为词项。
//
// Tokenizer 是 Analyzer 的核心组件，负责基本的文本分割。
// 分割后的词项可以进一步通过 Filter 处理。
type Tokenizer interface {
	// Tokenize 对文本进行分词。
	// fieldName 参数用于标记词项所属字段。
	Tokenize(fieldName string, text string) TokenStream
}

// TokenizerFunc 函数类型实现 Tokenizer 接口。
type TokenizerFunc func(fieldName string, text string) TokenStream

// Tokenize 实现了 Tokenizer 接口。
func (f TokenizerFunc) Tokenize(fieldName string, text string) TokenStream {
	return f(fieldName, text)
}

// StandardAnalyzer 是默认的分析器实现。
//
// StandardAnalyzer 提供以下功能：
//   - Unicode 文本分割（按单词边界）
//   - 转小写
//   - 过滤 ASCII 标点符号
//   - 过滤停用词（可选）
//   - 过滤超过最大长度的词项
//
// 适用于英文和其他西方语言的通用分析器。
type StandardAnalyzer struct {
	tokenizer  Tokenizer
	stopWords  map[string]struct{}
	maxWordLen int
	lowerCase  bool
}

// NewStandardAnalyzer 创建默认的 StandardAnalyzer。
//
// 默认配置：
//   - 启用小写转换
//   - 过滤长度 > 50 的词项
//   - 使用默认停用词表
func NewStandardAnalyzer() *StandardAnalyzer {
	return &StandardAnalyzer{
		tokenizer:  NewStandardTokenizer(),
		stopWords:  defaultStopWords(),
		maxWordLen: 50,
		lowerCase:  true,
	}
}

// StandardAnalyzerOption 配置 StandardAnalyzer 的选项。
type StandardAnalyzerOption func(*StandardAnalyzer)

// WithStopWords 设置停用词表。
func WithStopWords(words map[string]struct{}) StandardAnalyzerOption {
	return func(a *StandardAnalyzer) {
		a.stopWords = words
	}
}

// WithoutLowerCase 禁用小写转换。
func WithoutLowerCase() StandardAnalyzerOption {
	return func(a *StandardAnalyzer) {
		a.lowerCase = false
	}
}

// WithMaxWordLength 设置最大词项长度。
func WithMaxWordLength(max int) StandardAnalyzerOption {
	return func(a *StandardAnalyzer) {
		a.maxWordLen = max
	}
}

// NewStandardAnalyzerWithOpts 使用指定选项创建 StandardAnalyzer。
func NewStandardAnalyzerWithOpts(opts ...StandardAnalyzerOption) *StandardAnalyzer {
	a := &StandardAnalyzer{
		tokenizer:  NewStandardTokenizer(),
		stopWords:  defaultStopWords(),
		maxWordLen: 50,
		lowerCase:  true,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Analyze 实现了 Analyzer 接口。
func (a *StandardAnalyzer) Analyze(fieldName string, text string) TokenStream {
	// 基础分词
	stream := a.tokenizer.Tokenize(fieldName, text)

	// 应用过滤器链
	stream = NewLowerCaseFilter(stream, a.lowerCase)
	stream = NewStopWordFilter(stream, a.stopWords)
	stream = NewLengthFilter(stream, 2, a.maxWordLen)

	return stream
}

// DefaultField 实现了 Analyzer 接口。
func (a *StandardAnalyzer) DefaultField() string {
	return "_default"
}

// StandardTokenizer 是默认的分词器实现。
//
// StandardTokenizer 按 Unicode 单词边界进行分词：
//   - 连续的字母数字字符作为一个词项
//   - 跳过标点符号和其他分隔符
//   - 保留位置信息
type StandardTokenizer struct{}

// NewStandardTokenizer 创建新的 StandardTokenizer。
func NewStandardTokenizer() *StandardTokenizer {
	return &StandardTokenizer{}
}

// Tokenize 实现了 Tokenizer 接口。
func (t *StandardTokenizer) Tokenize(fieldName string, text string) TokenStream {
	if text == "" {
		return newBaseTokenStream(nil)
	}

	tokens := make([]*Token, 0, len(text)/3)
	position := 0

	runes := []rune(text)
	inWord := false
	wordStart := 0

	for i, r := range runes {
		isWordChar := isLetterOrNumber(r)

		if !inWord && isWordChar {
			// 开始一个新词
			inWord = true
			wordStart = i
		} else if inWord && !isWordChar {
			// 词结束
			if wordStart < i {
				tokens = append(tokens, &Token{
					Text:      string(runes[wordStart:i]),
					Start:     wordStart,
					End:       i,
					Position:  position,
					FieldName: fieldName,
					Type:      inferTokenType(string(runes[wordStart:i])),
				})
				position++
			}
			inWord = false
		}
	}

	// 处理最后一个词
	if inWord && wordStart < len(runes) {
		tokens = append(tokens, &Token{
			Text:      string(runes[wordStart:]),
			Start:     wordStart,
			End:       len(runes),
			Position:  position,
			FieldName: fieldName,
			Type:      inferTokenType(string(runes[wordStart:])),
		})
	}

	return newBaseTokenStream(tokens)
}

// isLetterOrNumber 检查是否为字母或数字。
func isLetterOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// inferTokenType 推断词项类型。
func inferTokenType(s string) TokenType {
	if len(s) == 0 {
		return TokenTypeWord
	}
	// 检查是否全部是数字
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return TokenTypeWord
		}
	}
	return TokenTypeNumber
}

// defaultStopWords 返回默认的停用词表。
func defaultStopWords() map[string]struct{} {
	stopWords := map[string]struct{}{
		"a": {}, "am": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
		"be": {}, "by": {}, "for": {}, "from": {}, "has": {},
		"he": {}, "i": {}, "in": {}, "is": {}, "it": {}, "its": {},
		"of": {}, "on": {}, "that": {}, "the": {}, "to": {},
		"was": {}, "were": {}, "will": {}, "with": {},
		"you": {}, "your": {}, "we": {}, "our": {}, "or": {},
		"not": {}, "this": {}, "but": {}, "if": {}, "then": {},
	}
	return stopWords
}
