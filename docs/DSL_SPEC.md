# DSL 规则说明书（V1）

## 1. 目标与原则

本说明书定义 Maure 查询 DSL 的规范语法与执行约束，作为后续开发与评审的统一标准。

原则：
1. 向后兼容：不破坏现有 `term/phrase/boolean/range/wildcard/fuzzy` 能力。
2. 可扩展：通过版本前缀、函数节点、作用域节点预留演进空间。
3. 可实现：语法规则可被当前 Go 解析器稳定落地。
4. 可测试：每条规则都可对应单元测试与集成测试。

---

## 2. DSL 总览

V1 支持：
1. 布尔查询：`AND` / `OR` / `NOT`。
2. 短语查询：`field:"foo bar"`。
3. 范围查询：`field:[a TO b]`、`field:{a TO b}`。
4. 通配符查询：`field:prefix*`（仅后缀 `*`）。
5. 模糊查询：`field:term~1`（仅编辑距离 1）。
6. 字段存在：`field:*`。
7. 查询作用域：`IN index("name")`（多索引入口）。
8. DSL 内分页：`LIMIT size` 或 `LIMIT from,size`。
9. DSL 内排序：`SORT BY field DESC`。

---

## 3. 语法定义（EBNF）

```ebnf
query          = [version] [require_in] [scope] expr [page] [sort] ;

version        = "@v" integer ;
require_in     = "REQUIRE_IN" ;
scope          = "IN" scope_item {"," scope_item} ;
scope_item     = ident "(" string_lit ")" ;

expr           = or_expr ;
or_expr        = and_expr { "OR" and_expr } ;
and_expr       = not_expr { "AND" not_expr } ;
not_expr       = [ "NOT" ] primary ;

primary        = term
               | phrase
               | range
               | wildcard
               | fuzzy
               | exists
               | func_call
               | "(" expr ")" ;

term           = [field ":"] value ;
phrase         = [field ":"] '"' { char } '"' ;
range          = field ":" ("[" | "{") value "TO" value ("]" | "}") ;
wildcard       = field ":" prefix "*" ;
fuzzy          = field ":" token "~" integer ;
exists         = field ":*" ;

func_call      = ident "(" [arg {"," arg}] ")" ;
arg            = value | field | func_call ;

page           = "LIMIT" integer ["," integer] ;
sort           = "SORT" "BY" sort_item {"," sort_item} ;
sort_item      = field [ "ASC" | "DESC" ] ;

field          = ident {"." ident} ;
value          = string_lit | number | datetime | boolean ;
ident          = letter { letter | digit | "_" | "-" } ;
```

---

## 4. 语义规则（必须遵循）

### 4.1 运算优先级
1. `NOT` > `AND` > `OR`。
2. 括号优先级最高。

### 4.2 字段与默认字段
1. 字段名大小写不敏感，内部统一小写。
2. 未显式指定字段时，默认字段为 `message`（可配置）。

### 4.3 范围查询
1. `[` `]` 为闭区间，`{` `}` 为开区间。
2. V1 仅支持数值与时间范围。
3. 字符串范围暂不支持（保留扩展）。

### 4.4 通配符查询
1. V1 仅支持 `prefix*`。
2. 禁止前导 `*abc`。
3. 禁止中缀模式 `a*b`。

### 4.5 模糊查询
1. V1 仅允许 `~1`。
2. `~0`、`~2+` 均视为非法输入。

### 4.6 分页与排序
1. `LIMIT size` 等价 `from=0,size=size`。
2. `LIMIT from,size` 表示偏移分页。
3. 若未显式排序，命中排序键固定为：`score DESC, docID ASC`（保证稳定分页）。

### 4.7 作用域（多索引）
1. `IN index("a")` 表示索引路由过滤。
2. 可写多个：`IN index("a"),index("b")`。
3. 若作用域不可用（未实现多索引路由），应返回明确错误而非静默忽略。

### 4.8 可选约束 REQUIRE_IN
1. `REQUIRE_IN` 为可选约束，声明后必须显式携带 `IN index("...")`。
2. 若声明 `REQUIRE_IN` 但缺少 `IN`，应返回明确错误。
3. 若声明 `REQUIRE_IN` 且执行器无法完成作用域过滤（例如文档无 `index` 字段），应返回明确错误。

---

## 5. 错误处理规范

解析或语义错误必须返回可读错误，至少包含：
1. 错误类型（`syntax_error` / `semantic_error`）。
2. 位置（token 或字符偏移）。
3. 建议修复方式。

推荐错误示例：
1. `syntax_error at token '~2': fuzzy distance only supports ~1 in v1`
2. `syntax_error near '*abc': leading wildcard is not supported in v1`
3. `semantic_error on field 'price': string range is not supported in v1`

---

## 6. 扩展机制（V2+ 预留）

### 6.1 版本前缀
1. `@v1` 为当前默认规范。
2. 未来新增规则使用 `@v2`，避免破坏旧查询。

### 6.2 函数扩展
1. 统一通过 `func_call` 语法挂载高级能力。
2. 未来可扩展：`match()`, `boost()`, `geo_distance()`, `rerank()`。

### 6.3 作用域扩展
1. 当前仅定义 `index()`。
2. 可扩展 `tenant()`, `cluster()`, `namespace()`。

---

## 7. 实现约束（工程规则）

1. Parser 层只负责语法树构建，不承担检索执行逻辑。
2. Query 执行层必须遵守统一排序与分页切片规则。
3. 不允许在多个查询类型中重复实现同一排序逻辑（避免漂移）。
4. 新增语法时必须补齐：
   - Parser 正反例测试
   - 执行层命中/边界测试
   - CLI/API 回归测试

---

## 8. 测试基线（每次 DSL 变更必跑）

1. `go test ./pkg/query/...`
2. `go test ./pkg/index/...`
3. `go test ./cmd/maure/command/...`
4. `go test ./...`
5. `go test -race ./...`

---

## 9. 示例（V1）

1. `@v1 level:error AND message:"db timeout"`
2. `@v1 price:[100 TO 300] AND title:iph*`
3. `@v1 timestamp:[2026-02-10T09:00:00Z TO 2026-02-10T10:00:00Z]`
4. `@v1 name:roam~1 OR name:foam~1`
5. `@v1 IN index("app"),index("ops") service:gateway LIMIT 20,50 SORT BY timestamp DESC`

---

## 10. 开发执行清单（以后按此照做）

新增 DSL 功能时，必须按以下顺序执行：
1. 在本文件补充语法和语义（先文档后代码）。
2. 定义 AST 变更与兼容策略。
3. 实现 parser 与错误信息。
4. 实现执行层与性能边界。
5. 补齐测试与基准。
6. 更新 `docs/CLI_API_REFERENCE.md` 示例。
7. 在 PR 描述中标注：语法变更、兼容影响、回滚方案。

---

## 11. DSL 模块架构（解耦版）

当前实现采用四层结构：
1. `pkg/dsl/contracts.go`：抽象层（Lexer/ParserDef/Validator/Planner/Compiler）。
2. `pkg/dsl/pipeline.go`：主体层（Engine 管线编排）。
3. `pkg/dsl/components/*.go`：组件层（term/phrase/range/wildcard/fuzzy/exists/boolean/meta）。
4. `pkg/query/dsl_adapter.go`：适配层（Plan -> QueryPlan/Query）。

扩展新语法时请遵循：
1. 在 `pkg/dsl/components/` 新增组件并注册到 `DefaultRegistry()`。
2. 在 `pkg/dsl/parser.go` 的组件结果映射中补充 AST 转换。
3. 在 `pkg/query/dsl_adapter.go` 注册并实现对应编译器。
4. 补齐组件测试、pipeline 测试、adapter 测试及回归样例。
