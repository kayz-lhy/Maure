package query

import (
	"maure/pkg/index"
	"strings"
)

// QueryParser 将查询字符串解析为查询对象。
//
// 支持的语法：
//   - 简单词项：hello
//   - AND 查询：hello AND world
//   - OR 查询：hello OR world
//   - NOT 查询：hello NOT world
//   - 组合查询：hello AND (world OR programming)
//   - 短语查询："hello world"
//   - 通配符：hello*（暂不支持）
//
// 优先级：NOT > AND > OR
type QueryParser struct{}

// NewQueryParser 创建新的查询解析器。
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse 将查询字符串解析为 Query。
func (p *QueryParser) Parse(s string) (Query, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}

	// 预处理：移除多余空格
	s = strings.TrimSpace(s)

	// 分割为词项列表
	tokens := p.tokenize(s)

	if len(tokens) == 0 {
		return nil, nil
	}

	// 解析查询
	parser := &simpleParser{tokens: tokens}
	return parser.Parse()
}

// tokenize 将查询字符串分割为词项。
func (p *QueryParser) tokenize(s string) []string {
	var tokens []string
	var current strings.Builder

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch c {
		case ' ', '\t', '\n':
			if current.Len() > 0 {
				tokens = append(tokens, p.normalizeKeyword(current.String()))
				current.Reset()
			}
		case '"':
			// 处理短语
			i++
			start := i
			for i < len(s) && s[i] != '"' {
				i++
			}
			if i > start {
				tokens = append(tokens, `"`+s[start:i]+`"`)
			}
		case '(', ')':
			if current.Len() > 0 {
				tokens = append(tokens, p.normalizeKeyword(current.String()))
				current.Reset()
			}
			tokens = append(tokens, string(c))
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, p.normalizeKeyword(current.String()))
	}

	return tokens
}

// normalizeKeyword 将关键字转换为大写，普通词项转换为小写。
func (p *QueryParser) normalizeKeyword(token string) string {
	upper := strings.ToUpper(token)
	if upper == "AND" || upper == "OR" || upper == "NOT" {
		return upper
	}
	// 普通词项转换为小写以匹配索引
	return strings.ToLower(token)
}

// simpleParser 是简单的递归下降解析器。
type simpleParser struct {
	tokens []string
	pos    int
}

// Parse 解析整个查询。
func (p *simpleParser) Parse() (Query, error) {
	query, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	return query, nil
}

// parseOr 解析 OR 表达式。
func (p *simpleParser) parseOr() (Query, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.hasMore() {
		token := p.peek()
		if token == "OR" {
			p.consume() // 跳过 OR
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			// 转换为 OR 查询
			if orQuery, ok := left.(*DisjunctionQuery); ok {
				orQuery.queries = append(orQuery.queries, right)
			} else {
				left = NewDisjunctionQuery(left, right)
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseAnd 解析 AND 表达式。
func (p *simpleParser) parseAnd() (Query, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.hasMore() {
		token := p.peek()
		if token == "AND" {
			p.consume() // 跳过 AND
			right, err := p.parseNot()
			if err != nil {
				return nil, err
			}
			// 转换为 AND 查询
			left = NewConjunctionQuery(left, right)
		} else if token == "NOT" {
			// 处理 NOT: A NOT B = BooleanQuery(MUST: A, MUST_NOT: B)
			p.consume() // 跳过 NOT
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			// 转换为布尔查询
			boolQuery := NewBooleanQuery()
			boolQuery.Add(left, OccurMust, 1.0)
			boolQuery.Add(right, OccurMustNot, 1.0)
			left = boolQuery
		} else if token != "OR" && token != ")" && token != "" {
			// 隐式 AND（相邻的词项）
			right, err := p.parseNot()
			if err != nil {
				return nil, err
			}
			left = NewConjunctionQuery(left, right)
		} else {
			break
		}
	}

	return left, nil
}

// parseNot 解析 NOT 表达式。
func (p *simpleParser) parseNot() (Query, error) {
	if p.peek() == "NOT" {
		p.consume() // 跳过 NOT
		subQuery, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		// 返回一个包装查询，用于过滤
		return &notQuery{subQuery: subQuery}, nil
	}
	return p.parsePrimary()
}

// parsePrimary 解析基本查询（词项或括号表达式）。
func (p *simpleParser) parsePrimary() (Query, error) {
	token := p.consume()

	if token == "" {
		return nil, nil
	}

	// 括号表达式
	if token == "(" {
		query, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek() != ")" {
			return nil, nil // 缺少右括号
		}
		p.consume() // 跳过 )
		return query, nil
	}

	// 短语查询
	if strings.HasPrefix(token, `"`) && strings.HasSuffix(token, `"`) {
		phrase := token[1 : len(token)-1]
		terms := strings.Fields(phrase)
		if len(terms) > 1 {
			return NewPhraseQuery(terms...), nil
		}
		return NewTermQuery(terms[0]), nil
	}

	// 词项查询
	return NewTermQuery(token), nil
}

// hasMore 检查是否还有更多词项。
func (p *simpleParser) hasMore() bool {
	return p.pos < len(p.tokens)
}

// peek 查看当前词项但不消费。
func (p *simpleParser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return ""
}

// consume 消费当前词项并返回。
func (p *simpleParser) consume() string {
	if p.pos < len(p.tokens) {
		token := p.tokens[p.pos]
		p.pos++
		return token
	}
	return ""
}

// notQuery 是 NOT 查询的实现。
type notQuery struct {
	subQuery Query
}

// Search 实现了 Query 接口。
func (q *notQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	// NOT 查询本身不返回结果，只用于过滤
	return nil, nil
}

// Explain 实现了 Query 接口。
func (q *notQuery) Explain(idx *index.RAMIndex) string {
	return "NOT(...)"
}

// MustQuery 是 MUST 查询的实现（类似于 BooleanQuery 中的 MUST）。
type MustQuery struct {
	query Query
}

// NewMustQuery 创建新的 MUST 查询。
func NewMustQuery(query Query) *MustQuery {
	return &MustQuery{query: query}
}

// Search 实现了 Query 接口。
func (q *MustQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	return q.query.Search(idx)
}

// Explain 实现了 Query 接口。
func (q *MustQuery) Explain(idx *index.RAMIndex) string {
	return "MUST(...)"
}

// ShouldQuery 是 SHOULD 查询的实现。
type ShouldQuery struct {
	queries []Query
}

// NewShouldQuery 创建新的 SHOULD 查询。
func NewShouldQuery(queries ...Query) *ShouldQuery {
	return &ShouldQuery{queries: queries}
}

// Search 实现了 Query 接口。
func (q *ShouldQuery) Search(idx *index.RAMIndex) ([]index.ScoreDoc, error) {
	results := make(map[int64]index.ScoreDoc)

	for _, subQuery := range q.queries {
		subResults, _ := subQuery.Search(idx)
		for _, r := range subResults {
			existing := results[r.DocID]
			if r.Score > existing.Score {
				results[r.DocID] = r
			}
		}
	}

	// 转换为切片
	resultSlice := make([]index.ScoreDoc, 0, len(results))
	for _, r := range results {
		resultSlice = append(resultSlice, r)
	}

	// 使用已有的 sortResults 函数
	sortResults(resultSlice)

	return resultSlice, nil
}

// Explain 实现了 Query 接口。
func (q *ShouldQuery) Explain(idx *index.RAMIndex) string {
	return "SHOULD(...)"
}
