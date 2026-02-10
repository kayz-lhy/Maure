# Maure 搜索引擎

一个轻量级、可嵌入部署的 Go 搜索引擎，支持日志检索与通用文本检索。

## 当前版本

- `v1.0.0`

## 核心能力

- 倒排索引（内存 + 文件持久化）
- Standard Analyzer（标准分词）
- 查询：Term / Boolean / Phrase
- 评分：TF-IDF / BM25
- CLI（基于 Cobra）
- HTTP API（`/search`、`/doc/:id`、`/stats` 等）
- 日志解析导入（JSON / Logback / auto）

## 快速开始

### 1. 构建

```bash
go build -o bin/maure ./cmd/maure
```

### 2. 创建索引并导入日志

```bash
./bin/maure init /tmp/maure-demo/index
./bin/maure --index /tmp/maure-demo/index parse-log --log-format=json /tmp/maure-demo/app.log
```

兼容写法（仍可用，但会提示弃用）：

```bash
./bin/maure --index /tmp/maure-demo/index parse-log /tmp/maure-demo/app.log --format=json
```

### 3. 搜索

推荐写法：

```bash
./bin/maure --index /tmp/maure-demo/index search --group=level "error OR timeout"
```

兼容写法（仍可用，但会提示弃用）：

```bash
./bin/maure --index /tmp/maure-demo/index search "error OR timeout" --group=level
```

### 4. 启动 API 服务

```bash
./bin/maure --index /tmp/maure-demo/index serve --port 8080
```

## CLI 命令

| 命令 | 说明 |
|------|------|
| `init <path>` | 初始化索引目录 |
| `open [path]` | 打开索引目录 |
| `info [path]` | 显示索引信息 |
| `stats [path]` | 显示索引统计 |
| `compact [path]` | 触发一次快照提交 |
| `delete <path>` | 删除索引目录 |
| `add <file>` | 添加单文件 |
| `add-dir <dir>` | 批量添加目录 |
| `import <file>` | 从 JSON 导入 |
| `parse-log <file>` | 解析日志并导入 |
| `delete-doc <id>` | 删除文档 |
| `list [path]` | 列出文档 |
| `search <query>` | 搜索 |
| `count <query>` | 统计匹配文档数 |
| `terms [path]` | 查看词项 |
| `similarity [bm25|tfidf]` | 查询/设置评分算法（CLI 展示） |
| `serve [path]` | 启动 HTTP 服务 |
| `version` | 版本信息 |

全局参数：

- `--index <path>` 索引目录
- `--format <text|json>` CLI 输出格式
- `--analyzer <standard|ram>` 分析器
- `-v, --verbose` 详细输出

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/` | API 信息 |
| `GET` | `/search?q=<query>&agg=count&group=level` | 搜索与聚合 |
| `GET` | `/doc/<id>` | 获取文档 |
| `GET` | `/stats` | 索引统计 |
| `POST` | `/add` | 添加文档 |
| `DELETE` | `/delete?id=<id>` | 删除文档 |

说明：当前删除接口为 `DELETE /delete?id=<id>`，不是 `DELETE /doc/<id>`。

## 前端示例

- 日志检索任务台：`examples/frontend/log-console/index.html`
- 使用说明：`examples/frontend/log-console/README.md`

## 文档导航

- 开发指南：`docs/DEVELOPMENT.md`
- 架构说明：`docs/ARCHITECTURE.md`
- 需求规格：`docs/REQUIREMENTS.md`
- 测试策略：`docs/TEST_STRATEGY.md`
- 技术决策：`docs/TECH_DECISIONS.md`
- CLI/API 速查：`docs/CLI_API_REFERENCE.md`
- AI Agent 使用指南：`docs/AI_AGENT_GUIDE.md`
- 性能分析报告：`docs/reports/search-api-performance-analysis.md`
