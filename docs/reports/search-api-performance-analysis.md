# 搜索接口性能分析报告

- 日期：2026-02-10
- 范围：`/search` HTTP 接口及其前后链路（查询执行、聚合、高亮、前端调用模式）
- 目标：识别主要性能瓶颈，给出分阶段优化路线

## 1. 结论摘要

当前性能瓶颈主要来自：

1. 查询执行未真正应用 `topN`，导致大量全量排序与全量结果计算。
2. 前端采用 N+1 请求模式（一次 `/search` + 多次 `/doc/:id`）。
3. 布尔查询执行中存在较多 map 分配与多轮合并，复杂查询下开销明显。
4. 高亮逻辑重复进行文本预处理（`ToLower` + `[]rune`），放大 CPU 使用。
5. 服务启动时全量导入 RAM 索引，数据规模增大后冷启动成本高。

## 2. 证据与瓶颈定位

### 2.1 `topN` 参数未生效（高优先级）

- HTTP 层调用：`/Users/kayz/Projects/Go/Maure/cmd/maure/command/serve.go:159`
  - `results, err := s.idx.Search(parsedQuery, 20)`
- 索引层实现：`/Users/kayz/Projects/Go/Maure/pkg/index/index.go:124`
  - `RAMIndex.Search` 直接 `return query.Search(idx)`，未使用 `n`。
- 词项查询执行：`/Users/kayz/Projects/Go/Maure/pkg/index/index.go:300`
  - `sort.Sort(ScoreDocs(results))` 对全部命中文档排序。

影响：命中量越大，CPU 与延迟增长越明显。

### 2.2 前端 N+1 请求（高优先级）

- 前端实现：`/Users/kayz/Projects/Go/Maure/examples/frontend/log-console/index.html:432`
  - 先请求 `/search`，随后对每条结果请求 `/doc/:id`。

影响：一次搜索变成 `1 + N` 次 HTTP，网络与服务端序列化开销叠加。

### 2.3 布尔查询中间结构分配较多（高优先级）

- 布尔查询执行：`/Users/kayz/Projects/Go/Maure/pkg/query/query.go:115`
  - MUST/SHOULD/MUST_NOT 多轮 map 构建与合并。

影响：复杂表达式下 GC 压力上升，吞吐下降。

### 2.4 高亮重复文本预处理（中优先级）

- 高亮入口：`/Users/kayz/Projects/Go/Maure/cmd/maure/command/highlight_helper.go:23`
  - 对文档字段与多个 term 循环匹配。
- 高亮匹配：`/Users/kayz/Projects/Go/Maure/pkg/highlight/highlighter.go:39`
  - 每次匹配都执行 `strings.ToLower` 与 `[]rune` 构造。

影响：长文本与多 term 查询时 CPU 开销明显。

### 2.5 服务冷启动全量建 RAM 索引（中优先级）

- 服务启动路径：`/Users/kayz/Projects/Go/Maure/cmd/maure/command/serve.go:50`
  - 启动阶段遍历全部文档，逐条 `ramIdx.Add`。

影响：数据量增大后，服务启动时延与内存占用增长显著。

## 3. 优化优先级建议（按收益/成本）

### P0（先做）

1. 让 `topN` 真正生效（查询执行层 Top-K）。
2. 新增 `/search` 可选返回文档摘要，前端去除 N+1 请求。
3. 布尔查询执行改为更少分配的合并策略（AND 先短列表，OR 增量合并）。

### P1（第二阶段）

1. 高亮预处理缓存（单文档字段只做一次标准化）。
2. 高亮范围策略优化（只对前 K 条执行高亮）。
3. 聚合阶段减少重复字段访问与类型转换。

### P2（第三阶段）

1. 冷启动改造：文档懒加载或可选 warmup。
2. 指标化与压测体系（p50/p95/p99、QPS、CPU、内存）常态化。

## 4. 分阶段实施计划

### 阶段 A：高收益最小改动

1. 查询层增加 Top-K 逻辑，避免全量排序。
2. `/search` 增加 `include_doc=true`（或 `fields=...`）返回轻量字段。
3. 前端改为单次 `/search` 渲染，去掉逐条 `/doc` 拉取。

验收标准：

1. 相同查询条件下，`/search` 平均耗时下降明显（目标 >= 30%）。
2. 前端每次检索 HTTP 请求数从 `1 + N` 降为 `1`。

### 阶段 B：查询执行优化

1. 布尔查询合并重构，减少 map 分配。
2. `MUST_NOT` 过滤改为更高效路径（按候选集过滤）。

验收标准：

1. OR/AND 混合查询 p95 延迟继续下降（目标 >= 20%）。

### 阶段 C：高亮与启动优化

1. 高亮预处理缓存和上限策略。
2. 启动路径优化（懒加载或索引快照直接服务）。

验收标准：

1. 首页冷启动时间显著缩短。
2. 高亮开启时延迟增量可控。

## 5. 风险与注意事项

1. 排序/Top-K 改造需保证结果一致性（同分排序规则要固定）。
2. `/search` 响应结构扩展要保持兼容（默认行为不变）。
3. 布尔查询执行改造需补充回归测试（AND/OR/NOT/短语混合场景）。

## 6. 建议新增测试

1. 性能基准测试（`pkg/query`、`pkg/index`）
   - 小规模、中规模、大规模索引数据集。
2. API 压测用例
   - `/search` 普通查询、布尔查询、带聚合、带高亮。
3. 回归正确性
   - topN 结果稳定、聚合一致、highlight 不回退。

## 7. 当前状态

该报告仅完成分析与规划，不包含优化代码实现。
