# Maure 增量开发记录

> 记录每个增量的开发内容、设计决策和实现细节

---

## 增量 1：文档结构与分词器 ✅

**完成日期**：2024-XX-XX

### 目标

实现搜索引擎的基础组件：文档数据结构和文本分析器。

### 完成内容

| 组件 | 路径 | 说明 |
|------|------|------|
| **document 包** | `pkg/document/document.go` | Document、Field、FieldType 数据结构 |
| **analyzer 包** | `pkg/analyzer/analyzer.go` | Analyzer、StandardAnalyzer 接口与实现 |
| | `pkg/analyzer/token.go` | Token、TokenStream 接口 |
| | `pkg/analyzer/filter.go` | LengthFilter、LowerCaseFilter |
| | `pkg/analyzer/stopword.go` | StopWordFilter |

### 核心设计

#### 文档结构

```go
// Document 是搜索引擎的基本存储单元
type Document struct {
    Fields []*Field   // 字段列表
    Boost  float32    // 文档权重
}

// Field 定义字段属性
type Field struct {
    Name      string      // 字段名
    Value     interface{} // 值
    FieldType FieldType   // 类型
    Indexed   bool        // 是否索引
    Tokenized bool        // 是否分词
    Stored    bool        // 是否存储
}
```

#### 分词器架构

```
输入文本 → Tokenizer → TokenStream → FilterChain → 输出 TokenStream
                                    ↑
                              Filter 1 (小写)
                              Filter 2 (停用词)
                              Filter 3 (长度)
```

#### 接口设计

```go
// Analyzer 接口 - 可扩展
type Analyzer interface {
    Analyze(fieldName, text string) TokenStream
    DefaultField() string
}

// Tokenizer 接口 - 可扩展
type Tokenizer interface {
    Tokenize(fieldName, text string) TokenStream
}

// TokenStream 接口 - 可迭代
type TokenStream interface {
    Next() bool
    Current() *Token
    Reset()
    Close()
}
```

### 测试覆盖

```
pkg/analyzer: 72.5%
pkg/document: 57.5%
```

### 演示测试

运行：`go test -v ./test/ -run TestDemo_DocumentAndAnalyzer`

### 可扩展点

| 扩展点 | 当前实现 | 未来可扩展 |
|--------|----------|------------|
| Analyzer | StandardAnalyzer | ChineseAnalyzer, NGramAnalyzer |
| Tokenizer | StandardTokenizer | EdgeNGramTokenizer |
| Filter | 3种 | 同义词、词根化、词性标注 |
| FieldType | 6种 | DateTime、GeoPoint |

---

## 增量 2：倒排索引与内存存储 ✅

**完成日期**：2024-XX-XX

### 目标

实现搜索引擎的核心数据结构：倒排索引，以及基于内存的存储实现。

### 完成内容

| 组件 | 路径 | 说明 |
|------|------|------|
| **store 包** | `pkg/store/store.go` | Directory、IndexWriter、IndexReader 接口 |
| | `pkg/store/ram_directory.go` | RAMDirectory 实现 |
| **index 包** | `pkg/index/index.go` | RAMIndex、Query、ScoreDoc |
| | `pkg/index/inverted.go` | InvertedIndex 核心数据结构 |

### 核心设计

#### 倒排索引结构

```
┌─────────────────────────────────────────────────────────┐
│                    InvertedIndex                         │
│  terms: map[string]*Postings                           │
│  - "fox" → Postings{                                   │
│      DocIDs: [1, 2],                                   │
│      Freqs: [1, 1],                                    │
│      Positions: [[0], [0]]                             │
│    }                                                    │
│  - "programming" → Postings{...}                       │
└─────────────────────────────────────────────────────────┘
```

#### 倒排表结构

```go
type Postings struct {
    DocIDs    []int64  // 文档 ID 列表
    Freqs     []int    // 词频
    Positions [][]int  // 位置列表（用于短语查询）
}
```

#### 存储接口抽象

```go
// Directory 接口 - 可扩展
type Directory interface {
    CreateIndexWriter() IndexWriter
    OpenIndexReader() IndexReader
    Exists(name string) bool
    // ...
}

// Index 接口 - 搜索引擎主接口
type Index interface {
    Add(doc *Document) (int64, error)
    Delete(docID int64) error
    Search(query Query, n int) []ScoreDoc
    DocCount() int64
    Close() error
}
```

### 词项查询实现

```go
// TermQuery - 精确词项查询
type TermQuery struct {
    Term  string
    Boost float32
}

func (q *TermQuery) Search(idx *RAMIndex) []ScoreDoc {
    postings, _ := idx.inverted.GetPostings(q.Term)
    // 返回匹配的文档列表
}
```

### 测试覆盖

```
pkg/index: 68.1%
```

### 演示测试

运行：`go test -v ./test/ -run TestDemo_IndexAndInvertedIndex`

### 可扩展点

| 扩展点 | 当前实现 | 未来可扩展 |
|--------|----------|------------|
| Directory | RAMDirectory | FSDirectory、MemoryMapDirectory |
| Scoring | BM25 | LMDirichlet、PL2 |
| Query | TermQuery | BooleanQuery、PhraseQuery |

---

## 增量 3：评分排序 ✅

**完成日期**：2024-XX-XX

### 目标

实现相关性评分算法，使搜索结果按相关性排序。

### 完成内容

| 组件 | 路径 | 说明 |
|------|------|------|
| **search 包** | `pkg/search/similarity.go` | Similarity 接口、TF-IDF、BM25 |

### 核心设计

#### 评分算法接口

```go
type Similarity interface {
    // 计算词项的评分
    Score(termFreq, docFreq, docLength int, avgDocLength float64, numDocs int64) float32
    Name() string
}
```

#### TF-IDF 公式

```
TF-IDF = sqrt(tf) * (log(N/df) + 1)

其中：
- tf: 词项在文档中的频率
- df: 包含该词项的文档数
- N:  索引中的总文档数
```

#### BM25 公式

```
BM25 = IDF * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl/avgdl))

其中：
- k1: 词频饱和参数（默认 1.2）
- b:  文档长度归一化参数（默认 0.75）
- dl: 当前文档长度
- avgdl: 平均文档长度
```

### 索引集成

```go
// RAMIndex 支持评分算法
type RAMIndex struct {
    similarity search.Similarity  // 评分算法
}

// 设置评分算法
idx.SetSimilarity(search.NewBM25Similarity())
idx.SetSimilarity(search.NewTFIDFSimilarity())
```

### 测试覆盖

```
pkg/search: 65.5%
pkg/index: 70.4%
pkg/query: 全部测试通过
```

### 演示测试

运行：`go test -v ./test/ -run TestDemo_Scoring`

### 可扩展点

| 扩展点 | 当前实现 | 未来可扩展 |
|--------|----------|------------|
| Similarity | TF-IDF、BM25 | LMDirichlet、PL2、DFR |
| 参数配置 | 固定/可选 | 可动态调整 |
| 多字段评分 | 简单求和 | 加权融合 |

---

## 增量 4：布尔查询 ✅

**完成日期**：2025-02-09

### 目标

实现布尔查询功能，支持 AND、OR、NOT 操作符，以及查询字符串解析。

### 完成内容

| 组件 | 路径 | 说明 |
|------|------|------|
| **query 包** | `pkg/query/query.go` | BooleanQuery、ConjunctionQuery、DisjunctionQuery |
| | `pkg/query/term_query.go` | TermQuery、PhraseQuery |
| | `pkg/query/parser.go` | 查询解析器（递归下降） |
| **演示测试** | `test/demo_query_test.go` | 查询功能演示 |

### 核心设计

#### 查询类型

```
┌─────────────────────────────────────────────────────────┐
│                     查询接口                            │
│  type Query interface {                                 │
│      Search(idx *RAMIndex) ([]ScoreDoc, error)          │
│      Explain(idx *RAMIndex) string                     │
│  }                                                     │
└─────────────────────────────────────────────────────────┘
         │
         ├── TermQuery        ← 精确词项匹配
         ├── PhraseQuery      ← 短语匹配（支持 slop）
         ├── ConjunctionQuery ← AND 查询（交集）
         ├── DisjunctionQuery ← OR 查询（并集）
         └── BooleanQuery     ← 布尔查询（混合 MUST/SHOULD/MUST_NOT）
```

#### BooleanQuery 结构

```go
type BooleanQuery struct {
    clauses []BooleanClause  // 查询子句
}

type BooleanClause struct {
    query Query     // 子查询
    occur Occur     // 发生条件
    boost float32   // 权重
}

type Occur int
const (
    OccurMust     Occur = iota  // 必须匹配（AND）
    OccurShould                  // 应该匹配（OR）
    OccurMustNot                 // 必须不匹配（NOT）
)
```

#### 查询解析器语法

```
查询 ::= 或表达式
或表达式 ::= 与表达式 { "OR" 与表达式 }
与表达式 ::= 非表达式 { ("AND" | 隐式AND) 非表达式 }
非表达式 ::= ["NOT"] 基本表达式
基本表达式 ::= 词项 | 短语 | "(" 查询 ")"
```

#### 支持的查询语法

| 语法 | 示例 | 说明 |
|------|------|------|
| 简单词项 | `go` | 精确匹配 |
| AND 查询 | `go AND programming` | 同时包含 |
| OR 查询 | `go OR java` | 包含任一 |
| NOT 查询 | `programming NOT java` | 包含前者不含后者 |
| 组合查询 | `(go OR python) AND programming` | 嵌套表达式 |
| 短语查询 | `"programming language"` | 连续词项 |

### 测试覆盖

```
pkg/query: 全部测试通过
- TermQuery 测试
- ConjunctionQuery 测试
- DisjunctionQuery 测试
- BooleanQuery 测试
- QueryParser 测试
- 短语查询测试
```

### 演示测试

运行：`go test -v ./test/ -run TestDemo_Query`

输出示例：
```
--- 查询: 'Go OR Java' ---
找到 3 个结果:
  DocID: 3, Score: 2.7102
  DocID: 1, Score: 2.1797
  DocID: 4, Score: 2.0234

--- 查询: '(Go OR Python) AND programming' ---
找到 2 个结果:
  DocID: 1, Score: 2.1797
  DocID: 4, Score: 2.0234
```

### 可扩展点

| 扩展点 | 当前实现 | 未来可扩展 |
|--------|----------|------------|
| QueryParser | 基本语法 | 支持通配符、正则 |
| BooleanQuery | MUST/SHOULD/MUST_NOT | minimum_should_match |
| PhraseQuery | 固定距离 slop | 多字段短语 |

---

## 项目整体状态

### 功能完成度

| 功能 | 状态 | 增量 |
|------|------|------|
| 文档添加 | ✅ 已完成 | 增量 1 |
| 分词器 | ✅ 已完成 | 增量 1 |
| 倒排索引 | ✅ 已完成 | 增量 2 |
| 词项查询 | ✅ 已完成 | 增量 2 |
| TF-IDF 评分 | ✅ 已完成 | 增量 3 |
| BM25 评分 | ✅ 已完成 | 增量 3 |
| 布尔查询 | ✅ 已完成 | 增量 4 |
| 短语查询 | ✅ 已完成 | 增量 5 |
| 文件持久化 | ✅ 已完成 | 增量 6 |

### 测试覆盖率

| 包 | 覆盖率 |
|----|--------|
| pkg/analyzer | 72.5% |
| pkg/document | 57.5% |
| pkg/index | 70.4% |
| pkg/search | 65.5% |
| **总计** | **~66%** |

> 目标：> 50%

### 性能指标

| 指标 | 目标 | 当前状态 |
|------|------|----------|
| 10万文档索引 | < 30秒 | 待测试 |
| 查询延迟 | < 100ms | 待测试 |
| 内存占用 | < 500MB | 待测试 |

---

## 开发规范

### 命名规范

- **包名**：简短、清晰、小写
- **公开类型**：驼峰命名、首字母大写
- **私有类型**：首字母小写
- **接口**：以 `er` 或 `or` 结尾

### 文档规范

- 公开 API 必须有注释
- 关键逻辑需要注释
- 包声明需说明用途
- 接口需说明行为

### 测试规范

- 每个包至少集成测试
- 演示测试放在 `test/` 目录
- 单元测试使用表驱动模式
- 覆盖率目标 > 50%

---

## 下一步

### 后续增量

7. CLI 工具集成
8. HTTP API 服务
9. 云原生部署

---

## 项目演进计划

### 当前形态：Go 依赖库

```
maure/                    ← 可作为 Go 依赖引入
├── pkg/
│   ├── analyzer/        ← import "maure/pkg/analyzer"
│   ├── document/
│   ├── index/
│   ├── query/           ← 新增：布尔查询
│   └── search/
└── go.mod: module maure
```

### 演进路径

```
依赖库 (当前) → v1.0 → v2.0
     │            │        │
     │            │        ├── 分布式搜索
     │            │        ├── 云原生部署
     │            │        └── 用户认证
     │            │
     │            ├── HTTP API 服务
     │            ├── Docker 部署
     │            └── Makefile
     │
     └── 已完成 ≈ 66%
```

### 演进方向

| 阶段 | 形态 | 功能 | 复杂度 |
|------|------|------|--------|
| **当前** | 依赖库 | 核心搜索功能 | 低 |
| **v1.0** | CLI + HTTP | 独立运行、服务部署 | 中 |
| **v2.0** | 云原生应用 | 分布式、认证、监控 | 高 |

### v1.0 规划

```
cmd/maure/              ← 新增 CLI 入口
├── main.go            ← 命令行入口 (~100 行)
└── commands/         ← 命令实现
    ├── init.go       # 初始化索引
    ├── add.go        # 添加文档
    ├── search.go     # 搜索命令
    └── serve.go      # HTTP 服务

api/                    ← 新增 API 层
└── handler.go        # HTTP 处理器

# 可实现
$ maure init           # 初始化索引
$ maure add file.txt  # 添加文档
$ maure search "query" # 搜索
$ maure serve --port 8080 # 启动 HTTP 服务
```

### v2.0 规划

```
# 高级功能
├── 分布式索引分片
├── 用户认证 (JWT)
├── 监控指标 (Prometheus)
├── 集群复制
└── 云原生部署 (K8s)
```

### 决策：先核心后工具

建议：先完成布尔查询、持久化等核心功能，再添加 CLI。

原因：
1. CLI 依赖核心功能的完整性
2. 核心功能稳定后再设计 API 更合理
3. 渐进式演进，风险更低
