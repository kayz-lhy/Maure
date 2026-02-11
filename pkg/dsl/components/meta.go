package components

import "strings"

var keywords = map[string]struct{}{
	"AND":   {},
	"OR":    {},
	"NOT":   {},
	"IN":    {},
	"LIMIT": {},
	"SORT":  {},
	"BY":    {},
	"ASC":   {},
	"DESC":  {},
	"TO":    {},
}

// NormalizeKeyword 标准化 DSL 关键字。
func NormalizeKeyword(token string) string {
	upper := strings.ToUpper(token)
	if _, ok := keywords[upper]; ok {
		return upper
	}
	return token
}
