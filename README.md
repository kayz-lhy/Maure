# Maure 搜索引擎

一个轻量级的 Go 语言搜索引擎，适合学习、研究和小规模应用。

## 特点

- **轻量级**：简单设计，资源占用低
- **易学习**：代码清晰，注释详细，适合学习搜索引擎原理
- **开箱即用**：依赖少，容易集成

## 快速开始

### 安装

```bash
git clone https://github.com/yourusername/maure.git
cd maure
go build -o bin/maure ./cmd/maure
```

### 基本使用

```go
package main

import (
    "fmt"
    "maure/pkg/analyzer"
    "maure/pkg/document"
    "maure/pkg/index"
)

func main() {
    // 创建分析器
    ana := analyzer.NewStandardAnalyzer()

    // 创建索引
    idx, err := index.NewRAMIndex(ana)
    if err != nil {
        panic(err)
    }
    defer idx.Close()

    // 添加文档
    doc := document.NewDocument()
    doc.Add(document.NewTextField("title", "Go 语言"))
    doc.Add(document.NewTextField("content", "Go 是一门简洁、高效的编程语言"))

    if err := idx.Add(doc); err != nil {
        panic(err)
    }

    // 搜索
    results, err := idx.Search("golang", 10)
    if err != nil {
        panic(err)
    }

    fmt.Printf("找到 %d 个结果\n", len(results))
}
```

## 功能

### 已支持

- [x] 文档添加
- [x] 文档删除/更新
- [x] Standard Analyzer（英文分词、小写转换、停用词过滤）
- [x] 倒排索引
- [x] 词项查询 (TermQuery)
- [x] TF-IDF 评分
- [x] BM25 评分
- [x] 内存索引 (RAMIndex)
- [ ] 布尔查询 (BooleanQuery)
- [ ] 短语查询 (PhraseQuery)
- [ ] 文件持久化

### 已完成增量

详见 [docs/INCREMENT.md](docs/INCREMENT.md)

| 增量 | 内容 | 状态 |
|------|------|------|
| 增量 1 | 文档结构与分词器 | ✅ |
| 增量 2 | 倒排索引与内存存储 | ✅ |
| 增量 3 | 评分排序 | 开发中 |

### 计划支持

- [ ] 布尔查询
- [ ] 短语查询
- [ ] TF-IDF / BM25 评分
- [ ] 范围查询
- [ ] 模糊查询
- [ ] 通配符查询
- [ ] 段合并
- [ ] 高亮显示
- [ ] 文件持久化

## 文档

- [开发指南](docs/DEVELOPMENT.md) - 快速上手
- [增量开发记录](docs/INCREMENT.md) - 开发历程
- [API 文档](docs/API.md) - 接口说明（待完善）
- [架构设计](docs/ARCHITECTURE.md) - 设计思路
- [需求规格](docs/REQUIREMENTS.md) - 功能列表

## 演示测试

```bash
go test -v ./test/                    # 运行所有演示测试
go test -v ./test/ -run TestDemo_DocumentAndAnalyzer  # 文档与分词器
go test -v ./test/ -run TestDemo_IndexAndInvertedIndex  # 倒排索引
```

## 项目结构

```
maure/
├── cmd/maure/          # CLI 工具（待开发）
├── pkg/
│   ├── analyzer/        # 分词器 ✅
│   ├── document/       # 文档结构 ✅
│   ├── index/          # 索引核心 ✅
│   ├── query/          # 查询（待开发）
│   ├── search/         # 评分（待开发）
│   └── store/          # 存储接口 ✅
├── test/               # 演示测试
├── docs/               # 文档
│   ├── INCREMENT.md    # 增量开发记录
│   ├── ARCHITECTURE.md # 架构设计
│   ├── REQUIREMENTS.md # 需求规格
│   └── DEVELOPMENT.md  # 开发指南
└── Makefile            # 构建脚本
```

## 构建与测试

```bash
make build    # 构建
make test     # 测试
make bench    # 基准测试
make lint     # 代码检查
```

## 开源协议

MIT License

## 贡献

欢迎 Issue 和 Pull Request！请先阅读 [贡献指南](CONTRIBUTING.md)。

## 为什么叫 Maure？

"Maure" 来自古法语，意为"黑暗、模糊"。搜索引擎的核心任务是从大量文本中找到相关信息，就像在黑暗中寻找光明。
