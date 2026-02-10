# AI Agent 使用指南

本文件用于让 AI Agent 在本项目中快速进入有效工作状态。

## 1. 项目定位

- 项目名：`Maure`
- 类型：轻量搜索引擎（非网站应用）
- 主要用途：日志检索、通用文本检索、可嵌入部署
- 核心交互：CLI + HTTP API

## 2. 技术栈与关键模块

- 语言：Go `1.24+`
- CLI：Cobra（`cmd/maure/cobra`）
- 业务命令：`cmd/maure/command`
- 核心包：
  - `pkg/index`：RAMIndex、倒排索引
  - `pkg/query`：查询解析与执行
  - `pkg/store`：FSDirectory、Snapshot、WAL
  - `pkg/logparser`：日志解析器
  - `pkg/aggregate`：count/group 聚合
  - `pkg/highlight`：高亮提取

## 3. 快速运行路径

```bash
# 构建
go build -o bin/maure ./cmd/maure

# 初始化索引
./bin/maure init /tmp/maure/index

# 导入日志
./bin/maure --index /tmp/maure/index parse-log --log-format=auto /tmp/maure/app.log

# 搜索
./bin/maure --index /tmp/maure/index search --group=level "error OR timeout"

# 启动 API
./bin/maure --index /tmp/maure/index serve --port 8080
```

## 4. HTTP API 事实（必须对齐）

- `GET /search?q=...&agg=count&group=...`
- `GET /doc/<id>`
- `GET /stats`
- `POST /add`
- `DELETE /delete?id=<id>`

注意：删除接口目前不是 `DELETE /doc/<id>`。

## 5. 当前兼容策略

- CLI 迁移到 Cobra 后仍保留旧写法兼容，并输出弃用提示：
  - `parse-log ... --format=json`（推荐改 `--log-format`）
  - `search "q" --group=level`（推荐 flag 在前）

## 6. 已知技术债（优先关注）

1. `topN` 未在查询执行层严格生效（存在全量排序风险）。
2. 前端示例当前为 N+1 请求模式（`/search` 后逐条 `/doc`）。
3. 复杂布尔查询 map 分配开销较高。

详见：`docs/reports/search-api-performance-analysis.md`。

## 7. 修改代码时的约束

1. 优先保持接口向后兼容。
2. 先补测试再改核心逻辑（特别是 `pkg/query` 与 `pkg/store`）。
3. 变更命令参数或 API 时必须同步更新：
   - `README.md`
   - `docs/CLI_API_REFERENCE.md`
   - 相关示例文档

## 8. 推荐测试清单

```bash
go test ./...
go test -race ./...
go vet ./...
```

若只改 CLI/文档：

```bash
go test ./cmd/maure/...
```

## 9. 分支约定

- 功能分支前缀：`codex/`
- 示例：`codex/docs-refresh`
