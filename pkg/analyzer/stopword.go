package analyzer

// StopWordFilter 过滤停用词。
//
// StopWordFilter 从词流中移除指定的停用词。
// 停用词是语言中常见但对搜索意义不大的词（如 "the", "is", "at" 等）。
type StopWordFilter struct {
	source    TokenStream
	stopWords map[string]struct{}
}

// NewStopWordFilter 创建新的 StopWordFilter。
func NewStopWordFilter(source TokenStream, stopWords map[string]struct{}) *StopWordFilter {
	return &StopWordFilter{
		source:    source,
		stopWords: stopWords,
	}
}

// Next 实现了 TokenStream 接口。
func (f *StopWordFilter) Next() bool {
	for f.source.Next() {
		token := f.source.Current()
		if !f.isStopWord(token.Text) {
			return true
		}
	}
	return false
}

// Current 实现了 TokenStream 接口。
func (f *StopWordFilter) Current() *Token {
	return f.source.Current()
}

// Reset 实现了 TokenStream 接口。
func (f *StopWordFilter) Reset() {
	f.source.Reset()
}

// Close 实现了 TokenStream 接口。
func (f *StopWordFilter) Close() {
	f.source.Close()
}

// SetSource 设置上游 TokenStream。
func (f *StopWordFilter) SetSource(source TokenStream) {
	f.source = source
}

// isStopWord 检查是否为停用词。
func (f *StopWordFilter) isStopWord(text string) bool {
	_, ok := f.stopWords[text]
	return ok
}
