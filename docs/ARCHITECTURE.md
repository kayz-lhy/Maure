# Maure 架构设计

## 1. 整体架构

```
┌─────────────────────────────────────────┐
│              应用层 (CLI/API)             │
├─────────────────────────────────────────┤
│               搜索层                      │
│         IndexSearcher / Query            │
├─────────────────────────────────────────┤
│               评分层                      │
│            Similarity                    │
├─────────────────────────────────────────┤
│               索引层                      │
│         InvertedIndex / IndexWriter      │
├─────────────────────────────────────────┤
│               分析层                      │
│            Analyzer / Tokenizer          │
├─────────────────────────────────────────┤
│               文档层                      │
│             Document / Field             │
└─────────────────────────────────────────┘
```

## 2. 包结构

```
pkg/
├── analyzer/      # 分词器
│   ├── analyzer.go
│   ├── token.go
│   └── standard.go
├── document/      # 文档
│   └── document.go
├── index/         # 索引
│   ├── index.go
│   └── inverted.go
├── query/         # 查询
│   ├── query.go
│   └── parser.go
├── search/        # 评分
│   └── similarity.go
└── store/         # 存储
    └── store.go
```

## 3. 核心数据结构

### Document

```go
type Document struct {
    Fields []*Field
    Boost  float32
}

type Field struct {
    Name      string
    Value     interface{}
    FieldType FieldType   // Text, Numeric, Date, Stored
    Stored    bool
    Indexed   bool
    Tokenized bool
}
```

### InvertedIndex

```go
type InvertedIndex struct {
    terms   map[string]*Postings
    nextDoc int64
}

type Postings struct {
    DocIDs    []int64
    Freqs     []int
    Positions [][]int
}
```

## 4. 接口设计

### Index 接口

```go
type Index interface {
    Add(doc *document.Document) error
    Delete(docID int64) error
    Search(query string, n int) ([]ScoreDoc, error)
    DocCount() int64
    Close() error
}
```

### Analyzer 接口

```go
type Analyzer interface {
    Analyze(field string, text string) TokenStream
}

type Tokenizer interface {
    Tokenize(field string, text string) TokenStream
}
```

## 5. 评分算法

### TF-IDF

```
score = tf * idf
tf = sqrt(termFreq)
idf = log(N / df)
```

### BM25

```
score = idf * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * fieldLength / avgLength))
```

## 6. 文件格式

```
index/
├── segments    # 段元数据
├── _0.tim      # 术语字典
├── _0.tip      # 术语索引
├── _0.doc      # 倒排表
└── _0.pos      # 位置信息
```

## 7. 简单性原则

本项目遵循以下简单性原则：

1. **先实现后优化**
2. **代码可读优先**
3. **避免过度设计**
4. **依赖最少原则**
