# Maure 搜索引擎

一个轻量级的 Go 语言搜索引擎，适合学习、研究和小规模应用。

## 特点

- **轻量级**：简单设计，资源占用低
- **易学习**：代码清晰，注释详细，适合学习搜索引擎原理
- **开箱即用**：依赖少，容易集成
- **完整 CLI**：提供命令行工具和 HTTP API 服务

## 快速开始

### 安装

```bash
git clone https://github.com/yourusername/maure.git
cd maure
make install
```

### CLI 基本使用

```bash
# 初始化索引
maure init ./myindex

# 添加文档
maure add ./docs/file.txt
maure add-dir ./docs

# 搜索
maure search "golang tutorial"

# 启动 HTTP 服务
maure serve --port 8080
```

### API 使用

```bash
# 搜索
curl "http://localhost:8080/search?q=golang"
curl "http://localhost:8080/search?q=error&include_doc=true"
curl "http://localhost:8080/search?q=error&fields=message,level,timestamp"

# 获取统计
curl http://localhost:8080/stats

# 添加文档
curl -X POST http://localhost:8080/add \
  -H "Content-Type: application/json" \
  -d '{"id":"doc1","fields":{"title":"Go语言","content":"Go是一门简洁的编程语言"}}'
```

## 功能

### 已支持

- [x] 文档添加/删除/更新
- [x] Standard Analyzer（英文分词、小写转换、停用词过滤）
- [x] 倒排索引
- [x] 词项查询 (TermQuery)
- [x] 布尔查询 (BooleanQuery)
- [x] 短语查询 (PhraseQuery)
- [x] TF-IDF / BM25 评分
- [x] 内存索引 (RAMIndex)
- [x] 文件持久化 (WAL + Snapshot)
- [x] 完整 CLI 工具
- [x] HTTP API 服务

### CLI 命令

| 命令 | 说明 |
|------|------|
| `init <path>` | 初始化索引目录 |
| `info [path]` | 显示索引信息 |
| `stats [path]` | 显示索引统计 |
| `compact [path]` | 优化/压缩索引 |
| `add <file>` | 添加文件到索引 |
| `add-dir <dir>` | 批量添加目录 |
| `import <file>` | 从 JSON 导入 |
| `delete-doc <id>` | 删除文档 |
| `list [path]` | 列出所有文档 |
| `search <query>` | 搜索文档 |
| `count <query>` | 统计匹配数 |
| `terms [prefix]` | 列出词项 |
| `serve [--port]` | 启动 HTTP API 服务 |

### 全局选项

| 选项 | 说明 |
|------|------|
| `--index <path>` | 索引目录 (默认当前目录) |
| `--format <fmt>` | 输出格式 (text/json) |
| `--analyzer <a>` | 分析器类型 |
| `-v` | 详细输出 |

### HTTP API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | API 信息 |
| GET | `/search?q=<query>` | 搜索文档 |
| GET | `/search?q=<query>&include_doc=true` | 搜索并返回文档摘要 |
| GET | `/search?q=<query>&fields=...` | 搜索并返回字段白名单 |
| GET | `/doc/:id` | 获取文档 |
| GET | `/stats` | 索引统计 |
| POST | `/add` | 添加文档 |
| DELETE | `/doc/:id` | 删除文档 |

## 已完成增量

详见 [docs/INCREMENT.md](docs/INCREMENT.md)

| 增量 | 内容 | 状态 |
|------|------|------|
| 增量 1 | 文档结构与分词器 | ✅ |
| 增量 2 | 倒排索引与内存存储 | ✅ |
| 增量 3 | 评分排序 | ✅ |
| 增量 4 | 查询解析器 | ✅ |
| 增量 5 | FSDirectory | ✅ |
| 增量 6 | 文件持久化 | ✅ |
| 增量 7 | CLI 工具 | ✅ |

## 文档

- [开发指南](docs/DEVELOPMENT.md) - 快速上手
- [CLI / API 参考](docs/CLI_API_REFERENCE.md) - 接口与参数速查
- [Top-K 优化记录](docs/TOPK_OPTIMIZATION.md) - 查询性能优化策略与基准
- [增量开发记录](docs/INCREMENT.md) - 开发历程
- [架构设计](docs/ARCHITECTURE.md) - 设计思路
- [需求规格](docs/REQUIREMENTS.md) - 功能列表

## 演示测试

```bash
make demo              # 运行所有演示测试
go test -v ./test/ -run TestDemo_Persistence  # 持久化测试
go test -v ./test/ -run TestDemo_Scoring      # 评分测试
```

## 项目结构

```
maure/
├── cmd/maure/          # CLI 工具
│   ├── main.go         # 主入口
│   └── command/        # 命令实现
│       ├── base.go     # 命令基类
│       ├── index.go    # 索引管理命令
│       ├── doc.go      # 文档操作命令
│       ├── search.go   # 搜索命令
│       └── serve.go    # HTTP 服务
├── pkg/
│   ├── analyzer/        # 分词器
│   ├── document/       # 文档结构
│   ├── index/          # 索引核心
│   ├── query/          # 查询解析
│   ├── search/         # 评分算法
│   └── store/          # 存储实现
├── test/               # 演示测试
├── docs/               # 文档
└── Makefile            # 构建脚本
```

## 构建与测试

```bash
make build    # 构建 CLI
make install  # 构建并安装
make test     # 单元测试
make demo     # 集成测试
make bench    # 基准测试
make lint     # 代码检查
make run      # 启动 HTTP 服务
```

## 开源协议

MIT License

## 贡献

欢迎 Issue 和 Pull Request！请先阅读 [贡献指南](CONTRIBUTING.md)。

## 为什么叫 Maure？

"Maure" 来自古法语，意为"黑暗、模糊"。搜索引擎的核心任务是从大量文本中找到相关信息，就像在黑暗中寻找光明。
