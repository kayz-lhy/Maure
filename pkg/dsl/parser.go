package dsl

import (
	"fmt"
	"maure/pkg/dsl/components"
	"strconv"
	"strings"
	"time"
)

// Parser 负责将 DSL token 解析为 AST。
type Parser struct {
	registry *components.Registry
}

// NewParser 创建 DSL 解析器。
func NewParser() *Parser {
	return &Parser{registry: components.DefaultRegistry()}
}

// Parse 兼容入口：直接解析字符串。
func (p *Parser) Parse(input string) (*ParsedQuery, error) {
	lexer := DefaultLexer{}
	tokens, err := lexer.Tokenize(strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}
	return p.ParseTokens(tokens)
}

// ParseTokens 实现 ParserDef 接口。
func (p *Parser) ParseTokens(tokens []Token) (*AST, error) {
	raw := make([]string, 0, len(tokens))
	for _, t := range tokens {
		raw = append(raw, t.Raw)
	}
	if len(raw) == 0 {
		return &ParsedQuery{}, nil
	}

	ps := &state{tokens: raw, registry: p.registry}
	out := &ParsedQuery{Version: 1}

	if v, ok, err := ps.parseVersion(); err != nil {
		return nil, err
	} else if ok {
		out.Version = v
	}
	if ps.peekUpper() == "REQUIRE_IN" {
		ps.consume()
		out.RequireIn = true
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
	tokens   []string
	pos      int
	registry *components.Registry
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
		switch {
		case upper == "AND":
			s.consume()
			right, err := s.parseNot()
			if err != nil {
				return nil, err
			}
			left = AndExpr{Left: left, Right: right}
		case upper == "NOT":
			s.consume()
			right, err := s.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = FilterNotExpr{Include: left, Exclude: right}
		case components.IsBooleanBoundary(upper):
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

	if len(tok) >= 2 && strings.HasPrefix(tok, `"`) && strings.HasSuffix(tok, `"`) {
		text := tok[1 : len(tok)-1]
		return PhraseExpr{Text: text}, nil
	}

	node, ok, err := parseFieldExpressionWithRegistry(tok, s.registry)
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
	return parseFieldExpressionWithRegistry(tok, components.DefaultRegistry())
}

func parseFieldExpressionWithRegistry(tok string, registry *components.Registry) (Expr, bool, error) {
	parts := strings.SplitN(tok, ":", 2)
	if len(parts) != 2 {
		return nil, false, nil
	}
	field := strings.TrimSpace(parts[0])
	expr := strings.TrimSpace(parts[1])
	if field == "" || expr == "" {
		return nil, false, fmt.Errorf("invalid field expression: %s", tok)
	}
	result, ok, err := registry.ParseFieldExpression(field, expr, tok)
	if err != nil || !ok {
		return nil, ok, err
	}
	return buildExprFromComponentResult(result)
}

func buildExprFromComponentResult(result components.FieldParseResult) (Expr, bool, error) {
	switch result.Kind {
	case components.ExprExists:
		return ExistsExpr{Field: result.Field}, true, nil
	case components.ExprPhrase:
		return PhraseExpr{Field: result.Field, Text: result.Text}, true, nil
	case components.ExprRange:
		kind := RangeValueNumber
		if result.ValueKind == "time" {
			kind = RangeValueTime
		}
		return RangeExpr{
			Field:     result.Field,
			Lower:     result.Lower,
			Upper:     result.Upper,
			Kind:      kind,
			Inclusive: result.Inclusive,
		}, true, nil
	case components.ExprWildcard:
		return WildcardExpr{Field: result.Field, Prefix: result.Text}, true, nil
	case components.ExprFuzzy:
		return FuzzyExpr{Field: result.Field, Term: result.Text, Distance: result.Distance}, true, nil
	case components.ExprTerm:
		return TermExpr{Field: result.Field, Value: result.Text}, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported component result kind: %s", result.Kind)
	}
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

// splitRangeBounds 供 DSL 包内部测试与兼容逻辑复用。
func splitRangeBounds(content string) (string, string, bool) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return "", "", false
	}
	toIdx := -1
	for i, f := range fields {
		if strings.EqualFold(f, "TO") {
			toIdx = i
			break
		}
	}
	if toIdx <= 0 || toIdx >= len(fields)-1 {
		return "", "", false
	}
	lower := strings.Join(fields[:toIdx], " ")
	upper := strings.Join(fields[toIdx+1:], " ")
	return lower, upper, true
}

// inferRangeKind 供 DSL 包内部测试与兼容逻辑复用。
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

// parseRangeTime 供 DSL 包内部测试与兼容逻辑复用。
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
