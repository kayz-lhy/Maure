package dsl

// Token 表示词法分析后的基础单元。
type Token struct {
	Raw string
}

// AST 是解析结果的语法树别名。
type AST = ParsedQuery

// Plan 是 DSL 管线中的统一计划对象。
type Plan struct {
	Version   int
	RequireIn bool
	ExprTree  Expr
	Scopes    []Scope
	Limit     *LimitClause
	Sort      []SortClause
	Warnings  []string
}

// Executable 是执行层可消费的抽象结果。
type Executable interface{}

// Lexer 定义词法分析能力。
type Lexer interface {
	Tokenize(input string) ([]Token, error)
}

// ParserDef 定义语法分析能力。
type ParserDef interface {
	ParseTokens(tokens []Token) (*AST, error)
}

// Validator 定义语义校验能力。
type Validator interface {
	Validate(ast *AST) error
}

// Planner 定义计划构建能力。
type Planner interface {
	Build(ast *AST) (*Plan, error)
}

// Compiler 定义执行体编译能力。
type Compiler interface {
	Compile(plan *Plan) (Executable, error)
}
