package components

import "strings"

// IsBooleanBoundary 判断 token 是否结束 AND 链。
func IsBooleanBoundary(token string) bool {
	upper := strings.ToUpper(token)
	switch upper {
	case "OR", ")", "LIMIT", "SORT", "":
		return true
	default:
		return false
	}
}
