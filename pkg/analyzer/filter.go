package analyzer

// LengthFilter 过滤长度不符合要求的词项。
//
// LengthFilter 根据最小和最大长度过滤词项：
//   - 过滤长度 < min 的词项
//   - 过滤长度 > max 的词项
type LengthFilter struct {
	source TokenStream
	minLen int
	maxLen int
}

// NewLengthFilter 创建新的 LengthFilter。
func NewLengthFilter(source TokenStream, min, max int) *LengthFilter {
	return &LengthFilter{
		source: source,
		minLen: min,
		maxLen: max,
	}
}

// Next 实现了 TokenStream 接口。
func (f *LengthFilter) Next() bool {
	for f.source.Next() {
		token := f.source.Current()
		l := len(token.Text)
		if l >= f.minLen && l <= f.maxLen {
			return true
		}
	}
	return false
}

// Current 实现了 TokenStream 接口。
func (f *LengthFilter) Current() *Token {
	return f.source.Current()
}

// Reset 实现了 TokenStream 接口。
func (f *LengthFilter) Reset() {
	f.source.Reset()
}

// Close 实现了 TokenStream 接口。
func (f *LengthFilter) Close() {
	f.source.Close()
}

// SetSource 设置上游 TokenStream。
func (f *LengthFilter) SetSource(source TokenStream) {
	f.source = source
}

// LowerCaseFilter 将词项转换为小写。
type LowerCaseFilter struct {
	source    TokenStream
	lowerCase bool
}

// NewLowerCaseFilter 创建新的 LowerCaseFilter。
//
// lowerCase 参数控制是否启用小写转换。
func NewLowerCaseFilter(source TokenStream, lowerCase bool) *LowerCaseFilter {
	return &LowerCaseFilter{
		source:    source,
		lowerCase: lowerCase,
	}
}

// Next 实现了 TokenStream 接口。
func (f *LowerCaseFilter) Next() bool {
	for f.source.Next() {
		token := f.source.Current()
		token.Text = f.process(token.Text)
		return true
	}
	return false
}

// Current 实现了 TokenStream 接口。
func (f *LowerCaseFilter) Current() *Token {
	return f.source.Current()
}

// Reset 实现了 TokenStream 接口。
func (f *LowerCaseFilter) Reset() {
	f.source.Reset()
}

// Close 实现了 TokenStream 接口。
func (f *LowerCaseFilter) Close() {
	f.source.Close()
}

// SetSource 设置上游 TokenStream。
func (f *LowerCaseFilter) SetSource(source TokenStream) {
	f.source = source
}

// process 对词项文本进行处理。
func (f *LowerCaseFilter) process(text string) string {
	if !f.lowerCase {
		return text
	}
	result := make([]byte, len(text))
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c - 'A' + 'a'
		} else {
			result[i] = c
		}
	}
	return string(result)
}
