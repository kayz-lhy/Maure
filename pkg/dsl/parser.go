package dsl

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parser 负责将 DSL 文本解析为 AST。
type Parser struct{}

// NewParser 创建 DSL 解析器。
func NewParser() *Parser {
	return &Parser{}
}

// Parse 解析完整 DSL（版本/作用域/表达式/分页/排序）。
func (p *Parser) Parse(s string) (*ParsedQuery, error) {
	tokens := tokenize(strings.TrimSpace(s))
	if len(tokens) == 0 {
		return &ParsedQuery{}, nil
	}

	ps := &state{tokens: tokens}
	out := &ParsedQuery{Version: 1}

	if v, ok, err := ps.parseVersion(); err != nil {
		return nil, err
	} else if ok {
		out.Version = v
	}

	scopes, err := ps.parseScope()
	if err != nil {
		return nil, err
	}
	out.Scopes = scopes

	expr, err := ps.parseOr()
	if err != nil {
		return nil, err
	}
	out.Expr = expr

	if ps.peekUpper() == "LIMIT" {
		limit, err := ps.parseLimit()
		if err != nil {
			return nil, err
		}
		out.Limit = limit
	}

	if ps.peekUpper() == "SORT" {
		sortItems, err := ps.parseSort()
		if err != nil {
			return nil, err
		}
		out.Sort = sortItems
	}

	if ps.hasMore() {
		return nil, fmt.Errorf("unexpected token: %s", ps.peek())
	}

	return out, nil
}

type state struct {
	tokens []string
	pos    int
}

func (s *state) parseVersion() (int, bool, error) {
	tok := s.peek()
	if !strings.HasPrefix(strings.ToLower(tok), "@v") {
		return 0, false, nil
	}
	s.consume()
	vStr := tok[2:]
	v, err := strconv.Atoi(vStr)
	if err != nil || v <= 0 {
		return 0, false, fmt.Errorf("invalid version token: %s", tok)
	}
	return v, true, nil
}

func (s *state) parseScope() ([]Scope, error) {
	if s.peekUpper() != "IN" {
		return nil, nil
	}
	s.consume()
	scopes := make([]Scope, 0, 2)
	for {
		tok := s.consume()
		if tok == "" {
			return nil, fmt.Errorf("scope item expected after IN")
		}
		scope, err := parseScopeItem(tok)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
		if s.peek() != "," {
			break
		}
		s.consume()
	}
	return scopes, nil
}

func parseScopeItem(tok string) (Scope, error) {
	open := strings.IndexByte(tok, '(')
	close := strings.LastIndexByte(tok, ')')
	if open <= 0 || close <= open+1 {
		return Scope{}, fmt.Errorf("invalid scope item: %s", tok)
	}
	kind := strings.ToLower(strings.TrimSpace(tok[:open]))
	arg := strings.TrimSpace(tok[open+1 : close])
	if !strings.HasPrefix(arg, `"`) || !strings.HasSuffix(arg, `"`) || len(arg) < 2 {
		return Scope{}, fmt.Errorf("scope value must be quoted: %s", tok)
	}
	value := arg[1 : len(arg)-1]
	if value == "" {
		return Scope{}, fmt.Errorf("scope value cannot be empty: %s", tok)
	}
	return Scope{Kind: kind, Value: value}, nil
}

func (s *state) parseLimit() (*LimitClause, error) {
	s.consume() // LIMIT
	tok := s.consume()
	if tok == "" {
		return nil, fmt.Errorf("LIMIT requires value")
	}
	if s.peek() == "," {
		s.consume()
		tok2 := s.consume()
		if tok2 == "" {
			return nil, fmt.Errorf("invalid LIMIT size")
		}
		tok = tok + "," + tok2
	}
	parts := strings.Split(tok, ",")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid LIMIT value: %s", tok)
	}
	if len(parts) == 1 {
		size, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("invalid LIMIT size: %s", tok)
		}
		return &LimitClause{From: 0, Size: size}, nil
	}
	from, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || from < 0 {
		return nil, fmt.Errorf("invalid LIMIT from: %s", tok)
	}
	size, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || size <= 0 {
		return nil, fmt.Errorf("invalid LIMIT size: %s", tok)
	}
	return &LimitClause{From: from, Size: size}, nil
}

func (s *state) parseSort() ([]SortClause, error) {
	s.consume() // SORT
	if s.peekUpper() != "BY" {
		return nil, fmt.Errorf("SORT must be followed by BY")
	}
	s.consume()

	items := make([]SortClause, 0, 2)
	for {
		field := s.consume()
		if field == "" {
			return nil, fmt.Errorf("SORT BY requires field")
		}
		item := SortClause{Field: field}
		if dir := s.peekUpper(); dir == "ASC" || dir == "DESC" {
			s.consume()
			item.Desc = dir == "DESC"
		}
		items = append(items, item)
		if s.peek() != "," {
			break
		}
		s.consume()
	}
	return items, nil
}

func (s *state) parseOr() (Expr, error) {
	left, err := s.parseAnd()
	if err != nil {
		return nil, err
	}
	for s.peekUpper() == "OR" {
		s.consume()
		right, err := s.parseAnd()
		if err != nil {
			return nil, err
		}
		left = OrExpr{Left: left, Right: right}
	}
	return left, nil
}

func (s *state) parseAnd() (Expr, error) {
	left, err := s.parseNot()
	if err != nil {
		return nil, err
	}

	for s.hasMore() {
		upper := s.peekUpper()
		switch upper {
		case "AND":
			s.consume()
			right, err := s.parseNot()
			if err != nil {
				return nil, err
			}
			left = AndExpr{Left: left, Right: right}
		case "NOT":
			s.consume()
			right, err := s.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = FilterNotExpr{Include: left, Exclude: right}
		case "OR", ")", "LIMIT", "SORT", "":
			return left, nil
		default:
			right, err := s.parseNot()
			if err != nil {
				return nil, err
			}
			left = AndExpr{Left: left, Right: right}
		}
	}

	return left, nil
}

func (s *state) parseNot() (Expr, error) {
	if s.peekUpper() == "NOT" {
		s.consume()
		sub, err := s.parsePrimary()
		if err != nil {
			return nil, err
		}
		return NotExpr{Sub: sub}, nil
	}
	return s.parsePrimary()
}

func (s *state) parsePrimary() (Expr, error) {
	tok := s.consume()
	if tok == "" {
		return nil, nil
	}
	if tok == "(" {
		expr, err := s.parseOr()
		if err != nil {
			return nil, err
		}
		if s.peek() != ")" {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		s.consume()
		return expr, nil
	}

	if strings.HasPrefix(tok, `"`) && strings.HasSuffix(tok, `"`) {
		text := tok[1 : len(tok)-1]
		return PhraseExpr{Text: text}, nil
	}

	node, ok, err := parseFieldExpression(tok)
	if err != nil {
		return nil, err
	}
	if ok {
		return node, nil
	}

	return TermExpr{Value: strings.ToLower(tok)}, nil
}

func (s *state) hasMore() bool {
	return s.pos < len(s.tokens)
}

func (s *state) peek() string {
	if s.pos >= len(s.tokens) {
		return ""
	}
	return s.tokens[s.pos]
}

func (s *state) peekUpper() string {
	return strings.ToUpper(s.peek())
}

func (s *state) consume() string {
	if s.pos >= len(s.tokens) {
		return ""
	}
	tok := s.tokens[s.pos]
	s.pos++
	return tok
}

func parseFieldExpression(tok string) (Expr, bool, error) {
	parts := strings.SplitN(tok, ":", 2)
	if len(parts) != 2 {
		return nil, false, nil
	}
	field := strings.TrimSpace(parts[0])
	expr := strings.TrimSpace(parts[1])
	if field == "" || expr == "" {
		return nil, false, fmt.Errorf("invalid field expression: %s", tok)
	}

	if strings.HasPrefix(expr, "[") || strings.HasPrefix(expr, "{") {
		inclusive := strings.HasPrefix(expr, "[")
		endBracket := "]"
		if !inclusive {
			endBracket = "}"
		}
		if !strings.HasSuffix(expr, endBracket) {
			return nil, false, fmt.Errorf("invalid range syntax: %s", tok)
		}
		content := expr[1 : len(expr)-1]
		upperContent := strings.ToUpper(content)
		idx := strings.Index(upperContent, " TO ")
		if idx < 0 {
			return nil, false, fmt.Errorf("range query must contain TO: %s", tok)
		}
		lower := strings.TrimSpace(content[:idx])
		upper := strings.TrimSpace(content[idx+4:])
		if lower == "" || upper == "" {
			return nil, false, fmt.Errorf("invalid range bounds: %s", tok)
		}
		kind, err := inferRangeKind(lower, upper)
		if err != nil {
			return nil, false, fmt.Errorf("invalid range query %s: %w", tok, err)
		}
		return RangeExpr{Field: field, Lower: lower, Upper: upper, Kind: kind, Inclusive: inclusive}, true, nil
	}

	if strings.Contains(expr, "?") {
		return nil, false, fmt.Errorf("wildcard '?' is not supported: %s", tok)
	}
	if strings.Contains(expr, "*") {
		if !strings.HasSuffix(expr, "*") || strings.Count(expr, "*") != 1 {
			return nil, false, fmt.Errorf("only suffix '*' wildcard is supported: %s", tok)
		}
		prefix := strings.TrimSuffix(expr, "*")
		if strings.TrimSpace(prefix) == "" {
			return nil, false, fmt.Errorf("wildcard prefix cannot be empty: %s", tok)
		}
		return WildcardExpr{Field: field, Prefix: prefix}, true, nil
	}

	if strings.Contains(expr, "~") {
		if !strings.HasSuffix(expr, "~1") {
			return nil, false, fmt.Errorf("only fuzzy distance ~1 is supported: %s", tok)
		}
		term := strings.TrimSuffix(expr, "~1")
		if strings.TrimSpace(term) == "" {
			return nil, false, fmt.Errorf("fuzzy term cannot be empty: %s", tok)
		}
		return FuzzyExpr{Field: field, Term: term, Distance: 1}, true, nil
	}

	// 普通 field:value 词项
	return TermExpr{Field: field, Value: strings.ToLower(expr)}, true, nil
}

func inferRangeKind(lower string, upper string) (RangeValueKind, error) {
	if _, err := strconv.ParseFloat(lower, 64); err == nil {
		if _, err := strconv.ParseFloat(upper, 64); err == nil {
			return RangeValueNumber, nil
		}
	}

	if _, err := parseRangeTime(lower); err == nil {
		if _, err := parseRangeTime(upper); err == nil {
			return RangeValueTime, nil
		}
	}

	return 0, fmt.Errorf("range supports only numeric/time bounds")
}

func parseRangeTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

func tokenize(s string) []string {
	if s == "" {
		return nil
	}

	tokens := make([]string, 0, 16)
	var current strings.Builder
	inQuote := false
	inRange := false
	inFuncCall := false
	var rangeEnd byte

	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, normalizeKeyword(current.String()))
		current.Reset()
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inFuncCall {
			current.WriteByte(ch)
			if ch == ')' {
				inFuncCall = false
				flush()
			}
			continue
		}
		if inQuote {
			current.WriteByte(ch)
			if ch == '"' {
				inQuote = false
				flush()
			}
			continue
		}
		if inRange {
			current.WriteByte(ch)
			if ch == rangeEnd {
				inRange = false
				flush()
			}
			continue
		}

		switch ch {
		case ' ', '\t', '\n', '\r':
			flush()
		case '(':
			if current.Len() > 0 && !strings.Contains(current.String(), ":") {
				current.WriteByte(ch)
				inFuncCall = true
				continue
			}
			flush()
			tokens = append(tokens, "(")
		case ')':
			flush()
			tokens = append(tokens, ")")
		case ',':
			flush()
			tokens = append(tokens, ",")
		case '"':
			if current.Len() > 0 && strings.HasSuffix(current.String(), ":") {
				current.WriteByte(ch)
				i++
				start := i
				for i < len(s) && s[i] != '"' {
					i++
				}
				if i > start {
					current.WriteString(s[start:i])
				}
				if i < len(s) && s[i] == '"' {
					current.WriteByte('"')
				}
				flush()
				continue
			}
			flush()
			current.WriteByte(ch)
			inQuote = true
		case '[', '{':
			current.WriteByte(ch)
			if strings.Contains(current.String(), ":") {
				inRange = true
				rangeEnd = ']'
				if ch == '{' {
					rangeEnd = '}'
				}
			}
		default:
			current.WriteByte(ch)
		}
	}
	flush()

	return tokens
}
