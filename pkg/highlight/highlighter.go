package highlight

import "strings"

// Fragment 表示命中的高亮片段。
type Fragment struct {
	Text  string
	Start int
	End   int
}

// Highlighter 负责提取匹配位置与片段。
type Highlighter struct {
	fragmentSize int
	numFragments int
}

const (
	defaultFragmentSize = 160
	defaultNumFragments = 1
)

// NewHighlighter 创建默认配置的高亮器。
func NewHighlighter() *Highlighter {
	return &Highlighter{
		fragmentSize: defaultFragmentSize,
		numFragments: defaultNumFragments,
	}
}

// FindTerm 查找词项在文本中的首个字符范围（end 为开区间）。
func (h *Highlighter) FindTerm(text string, term string) (start, end int, ok bool) {
	textRunes := []rune(text)
	termRunes := []rune(strings.TrimSpace(term))
	if len(textRunes) == 0 || len(termRunes) == 0 {
		return 0, 0, false
	}

	lowerText := []rune(strings.ToLower(text))
	lowerTerm := []rune(strings.ToLower(term))
	window := len(lowerTerm)

	for i := 0; i+window <= len(lowerText); i++ {
		match := true
		for j := 0; j < window; j++ {
			if lowerText[i+j] != lowerTerm[j] {
				match = false
				break
			}
		}
		if match {
			return i, i + window, true
		}
	}
	return 0, 0, false
}

// Extract 提取首个匹配片段，不命中时返回 ok=false。
func (h *Highlighter) Extract(text string, term string) (Fragment, bool) {
	start, end, ok := h.FindTerm(text, term)
	if !ok {
		return Fragment{}, false
	}

	runes := []rune(text)
	half := h.fragmentSize / 2
	fragmentStart := start - half
	if fragmentStart < 0 {
		fragmentStart = 0
	}
	fragmentEnd := fragmentStart + h.fragmentSize
	if fragmentEnd < end {
		fragmentEnd = end
	}
	if fragmentEnd > len(runes) {
		fragmentEnd = len(runes)
	}
	if fragmentEnd-fragmentStart > h.fragmentSize && h.fragmentSize > 0 {
		fragmentEnd = fragmentStart + h.fragmentSize
	}

	return Fragment{
		Text:  string(runes[fragmentStart:fragmentEnd]),
		Start: start,
		End:   end,
	}, true
}
