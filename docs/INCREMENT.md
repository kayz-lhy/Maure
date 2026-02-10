# Maure 增量开发记录

> 本文档记录已落地的功能增量与关键决策，作为发布回溯与后续规划基线。

## 文档约定

- 状态定义：`已完成`、`进行中`、`计划中`
- 记录原则：仅写“已合入主线/可复现验证”的内容
- 验证口径：优先 `go test ./...`、`go vet ./...`、关键场景手工联调

---

## 增量总览

| 增量 | 主题 | 状态 | 关键交付 |
|---|---|---|---|
| M1 | 文档模型与分词器 | 已完成 | `document` / `analyzer` 基础能力 |
| M2 | 倒排索引与内存索引 | 已完成 | `InvertedIndex`、`RAMIndex` |
| M3 | 相关性评分 | 已完成 | TF-IDF / BM25 |
| M4 | 查询系统（基础） | 已完成 | Term / Boolean / Phrase + Parser |
| M5 | 持久化存储 | 已完成 | WAL + Snapshot + FSDirectory |
| M6 | CLI 工具化 | 已完成 | Cobra 命令树与兼容层 |
| M7 | HTTP 服务化 | 已完成 | `/search`、`/doc`、`/stats`、`/add`、`/delete` |
| M8 | 日志解析导入 | 已完成 | `parse-log` 与多格式适配 |
| M9 | 搜索聚合能力 | 已完成 | `count` + `group`（CLI/HTTP） |
| M10 | 高级查询 | 已完成 | 范围/通配（前缀）/模糊（~1） |
| M11 | 分页与 Top-K 语义修复（issue #39） | 已完成 | `from/size` + `from+size` 有界收集 |
| M12 | `/search` 文档摘要回填（issue #40） | 已完成 | `include_doc` / `fields`，消除前端 1+N |
| M13 | Top-K 性能优化 | 已完成 | 有界堆收集、BM25 上界剪枝、基准稳定化 |
| M14 | 发布与工程卫生 | 已完成 | 版本 `v1.1.0`、`.gitignore` 加固 |

---

## 详细增量记录

### M1 文档模型与分词器（已完成）

**目标**
- 建立统一文档表示与可扩展分析链路。

**交付**
- `pkg/document`：`Document`、`Field`、字段类型与访问接口。
- `pkg/analyzer`：标准分词、大小写归一、停用词与过滤器链。

**关键决策**
- 分析器采用接口抽象，后续可替换/扩展。

---

### M2 倒排索引与内存索引（已完成）

**目标**
- 提供可查询的内存索引核心。

**交付**
- `pkg/index/inverted.go`：倒排表、词频、位置存储。
- `pkg/index/index.go`：`RAMIndex` 增删改查、排序结果输出。

**关键决策**
- 位置列表入索引，为短语查询和高亮提供基础。

---

### M3 相关性评分（已完成）

**目标**
- 提供可切换的相关性评分能力。

**交付**
- `pkg/search/similarity.go`：TF-IDF 与 BM25。

**关键决策**
- 评分算法通过 `Similarity` 抽象注入，避免索引层硬编码。

---

### M4 查询系统（基础）（已完成）

**目标**
- 支持基础可组合查询语法。

**交付**
- `pkg/query/query.go`、`pkg/query/term_query.go`、`pkg/query/parser.go`。
- 支持：Term / Boolean（AND OR NOT）/ Phrase / 括号优先级。

**关键决策**
- 解析器采用递归下降，便于后续语法扩展。

---

### M5 持久化存储（已完成）

**目标**
- 索引可落盘恢复。

**交付**
- `pkg/store`：FSDirectory、WAL、Snapshot、读写器。

**关键决策**
- 使用快照 + 日志恢复模型，兼顾一致性与实现复杂度。

---

### M6 CLI 工具化（已完成）

**目标**
- 提供稳定 CLI 入口与命令组织。

**交付**
- `cmd/maure/cobra/*`：根命令与子命令分层。
- 兼容历史调用方式并提供弃用提示。

**关键决策**
- 解析与路由迁移到 Cobra，业务执行复用 `command` 层。

---

### M7 HTTP 服务化（已完成）

**目标**
- 提供可嵌入的检索 API。

**交付**
- `cmd/maure/command/serve.go`：
  - `GET /search`
  - `GET /doc/<id>`
  - `GET /stats`
  - `POST /add`
  - `DELETE /delete?id=<id>`

**关键决策**
- API 保持轻量，参数语义与 CLI 一致。

---

### M8 日志解析导入（已完成）

**目标**
- 让日志场景可直接入索引。

**交付**
- `pkg/logparser/*` + `parse-log` 命令。
- 支持 `json` / `logback` / `auto`。

---

### M9 搜索聚合能力（已完成）

**目标**
- 在检索结果上提供轻量聚合。

**交付**
- `pkg/aggregate/*`
- `search --agg=count --group=<field|time(...)>`
- HTTP `/search` 同步支持聚合参数。

---

### M10 高级查询（已完成）

**目标**
- 扩展字段级查询能力。

**交付**
- 范围查询：数值 / 时间。
- 通配符：前缀 `*`（prefix-only）。
- 模糊：编辑距离 `~1`。

**关键决策**
- 首版限制语法边界，优先正确性与可维护性。

---

### M11 分页与 Top-K 语义修复（issue #39，已完成）

**目标**
- 修复“假分页/全量排序”问题。

**交付**
- `/search` 支持 `from/size`。
- 执行层按 `from+size` 做有界收集与切片。
- 排序稳定键：`score desc, docID asc`。

**结果**
- 分页无重复无漏项，延迟随命中量增长更可控。

---

### M12 `/search` 文档摘要回填（issue #40，已完成）

**目标**
- 避免前端 `search + N次 /doc` 的 1+N 请求。

**交付**
- `/search` 新增参数：
  - `include_doc=true`
  - `fields=message,level,...`
- 命中项可返回：
  - `doc.summary`
  - `doc.fields`（白名单投影）

**结果**
- 前端检索详情可降为单请求。

---

### M13 Top-K 性能优化（已完成）

**目标**
- 降低高频词检索开销并稳定基准结果。

**交付**
- Top-K collector 路径减少额外分配。
- BM25 上界剪枝（堆满后跳过无机会候选）。
- 基准测试稳定化：预热、单线程口径、噪声控制。

**补充文档**
- `docs/TOPK_OPTIMIZATION.md`
- `docs/reports/search-api-performance-analysis.md`

---

### M14 发布与工程卫生（已完成）

**交付**
- 版本提升至 `v1.1.0`。
- `.gitignore` 加固（本地缓存/工作区文件忽略）。
- 文档一致性修订（CLI/API/README 对齐）。

---

## 当前能力边界（v1.1.0）

### 已具备
- 轻量嵌入式检索内核（RAM + FS）。
- CLI / HTTP 双入口。
- 日志解析导入、聚合统计、高亮输出。
- 高级查询与分页语义闭环。

### 已知限制
- 未提供分布式分片/副本。
- 未提供多租户鉴权与权限模型。
- 通配符/模糊查询仍是受限首版语法。

---

## 后续增量建议（计划中）

1. **查询执行器深度优化**：布尔查询有界合并、跳表/块级优化。  
2. **索引维护能力**：段合并策略、后台 compaction、冷热分层。  
3. **服务化能力**：可观测性指标、健康检查、限流与熔断。  
4. **发布工程化**：release checklist、changelog 自动生成、兼容矩阵。  

---

## 变更维护规则

- 新增功能必须在本文件追加“增量记录”，禁止只改状态表不写细节。  
- 涉及接口变更时，必须同步更新：
  - `README.md`
  - `docs/CLI_API_REFERENCE.md`
- 涉及性能优化时，必须同步更新：
  - `docs/TOPK_OPTIMIZATION.md` 或 `docs/reports/*`
