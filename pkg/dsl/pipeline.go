package dsl

import "fmt"

// Engine 负责 DSL 管线编排。
type Engine struct {
	lexer     Lexer
	parser    ParserDef
	validator Validator
	planner   Planner
	compiler  Compiler
}

// NewEngine 创建 DSL 管线引擎。
func NewEngine(lexer Lexer, parser ParserDef, validator Validator, planner Planner, compiler Compiler) *Engine {
	return &Engine{
		lexer:     lexer,
		parser:    parser,
		validator: validator,
		planner:   planner,
		compiler:  compiler,
	}
}

// BuildPlan 构建 DSL 计划。
func (e *Engine) BuildPlan(input string) (*Plan, error) {
	if e == nil {
		return nil, fmt.Errorf("dsl engine is nil")
	}
	if e.lexer == nil || e.parser == nil || e.validator == nil || e.planner == nil {
		return nil, fmt.Errorf("dsl engine has incomplete pipeline")
	}

	tokens, err := e.lexer.Tokenize(input)
	if err != nil {
		return nil, err
	}
	ast, err := e.parser.ParseTokens(tokens)
	if err != nil {
		return nil, err
	}
	if err := e.validator.Validate(ast); err != nil {
		return nil, err
	}
	return e.planner.Build(ast)
}

// BuildExecutable 构建可执行对象。
func (e *Engine) BuildExecutable(input string) (Executable, error) {
	if e == nil {
		return nil, fmt.Errorf("dsl engine is nil")
	}
	if e.compiler == nil {
		return nil, fmt.Errorf("dsl engine compiler is nil")
	}
	plan, err := e.BuildPlan(input)
	if err != nil {
		return nil, err
	}
	return e.compiler.Compile(plan)
}

// DefaultLexer 是默认词法实现。
type DefaultLexer struct{}

// Tokenize 将输入转换为 token 列表。
func (DefaultLexer) Tokenize(input string) ([]Token, error) {
	raw := tokenize(input)
	tokens := make([]Token, 0, len(raw))
	for _, t := range raw {
		tokens = append(tokens, Token{Raw: t})
	}
	return tokens, nil
}

// NoopValidator 是默认校验实现（首版仅占位）。
type NoopValidator struct{}

// Validate 实现 Validator。
func (NoopValidator) Validate(ast *AST) error {
	_ = ast
	return nil
}

// DefaultPlanner 是默认计划构建实现。
type DefaultPlanner struct{}

// Build 将 AST 映射为 Plan。
func (DefaultPlanner) Build(ast *AST) (*Plan, error) {
	if ast == nil {
		return &Plan{}, nil
	}
	return &Plan{
		Version:   ast.Version,
		RequireIn: ast.RequireIn,
		ExprTree:  ast.Expr,
		Scopes:    ast.Scopes,
		Limit:     ast.Limit,
		Sort:      ast.Sort,
	}, nil
}
