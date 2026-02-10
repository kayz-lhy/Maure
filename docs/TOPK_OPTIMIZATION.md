# Top-K 查询优化技术文档

## 1. 背景

在 `issue #39` 与后续性能回归中，我们观察到以下问题：

1. `top10` 与 `top1000` 延迟接近，说明 Top-K 限制未在“收集阶段”充分生效。
2. 高频词查询随文档量增大退化明显，热点查询成本偏高。
3. 堆操作与评分计算在热点词场景产生额外 CPU 与内存压力。

本轮优化目标是：

1. 让 Top-K 更早生效，减少无效候选保留。
2. 减少每文档重复评分计算的常量开销。
3. 降低热点查询的分配次数与分配体积。

---

## 2. 优化策略总览

### 策略 A：`SearchTopN` 快路径

在索引层加入可选快路径：

1. `RAMIndex.Search(query, n)` 检查查询对象是否实现 `SearchTopN(idx, n)`。
2. 若实现则直接走有界收集逻辑。
3. 未实现则兼容旧路径：`query.Search(idx)` 后排序截断。

这样可以逐个查询类型渐进优化，不破坏现有接口契约。

### 策略 B：词项查询有界 Top-K 收集

为 `TermQuery` 引入最小堆收集器：

1. 容量固定为 `n`。
2. 候选只在“优于当前堆顶最差项”时替换。
3. 最终仅对堆内结果排序输出。

该策略直接削减高频词场景下结果维护成本。

### 策略 C：评分常量预计算

在 `TermQuery` 内对 BM25/TF-IDF 做常量预计算：

1. `idf` 由每文档重复计算改为每查询计算一次。
2. BM25 参数 `k1/b` 与公式中的不变量提前提取。
3. 保留对其他 Similarity 的兼容 fallback。

该策略降低热点循环中的 `log/sqrt` 重复开销。

### 策略 D：稳定排序规则

统一排序规则为：

1. `score` 降序
2. 同分时 `docID` 升序

用于减少分页与 Top-K 的同分抖动，提升结果稳定性。

---

## 3. 代码改动点

### 索引层

1. `/Users/kayz/Projects/Go/Maure/pkg/index/index.go`
   - `RAMIndex.Search` 增加 `SearchTopN` 快路径。
   - `TermQuery` 支持 `searchInternal(idx, n)`。
   - 引入无接口分配的 `topKCollector`（最小堆）。
   - 加入 `better/worse` 比较函数与稳定排序辅助。

### 查询层

1. `/Users/kayz/Projects/Go/Maure/pkg/query/term_query.go`
   - 增加 `SearchTopN` 与 `searchInternal(idx, n)`。
   - 引入 `queryTopKCollector`。
   - 加入 BM25/TF-IDF 常量预计算快路径。

2. `/Users/kayz/Projects/Go/Maure/pkg/query/query.go`
   - `sortResults` 改为稳定排序（同分 `docID` 升序）。

### 评分层

1. `/Users/kayz/Projects/Go/Maure/pkg/search/similarity.go`
   - `BM25Similarity` 新增 `B()` 方法，支持查询层快路径参数读取。

### 基准测试

1. `/Users/kayz/Projects/Go/Maure/pkg/index/index_test.go`
   - 扩展 `BenchmarkRAMIndex_SearchLargeDataset`。
   - 覆盖 `10k/50k`、`hot/rare/unique`、`top10/top100/top1000`。

---

## 4. 基准测试方法

推荐命令：

```bash
go test -run '^$' -bench BenchmarkRAMIndex_SearchLargeDataset -benchmem ./pkg/index
```

针对热点对比（重点观察 `top10` vs `top1000`）：

```bash
go test -run '^$' -bench 'BenchmarkRAMIndex_SearchLargeDataset/50k/hot-top(10|1000)$' -benchmem -count=5 ./pkg/index
```

---

## 5. 本轮观测结果（Apple M2，Go benchmark）

单次热点对比（50k）：

1. `hot-top10`：约 `3.55 ms/op`，`803161 B/op`，`4 allocs/op`
2. `hot-top1000`：约 `3.83 ms/op`，`835608 B/op`，`4 allocs/op`

全量样本中（50k）：

1. `hot-top10`：约 `3.33 ms/op`
2. `hot-top100`：约 `4.04 ms/op`
3. `hot-top1000`：约 `9.74 ms/op`

多次样本（`-count=5`）显示存在波动，建议用 `benchstat` 统计显著性。

---

## 6. 效果评估

已达成：

1. Top-K 收集逻辑在词项查询路径中前置生效。
2. `allocs/op` 显著下降（尤其大 K 路径）。
3. 排序稳定性增强，分页一致性更好。

仍待提升：

1. 高频词场景仍需遍历和打分大量候选，`top10` 与大 K 差距在部分样本中仍不稳定。
2. 当前优化主要覆盖 `TermQuery`，布尔组合查询仍有进一步剪枝空间。

---

## 7. 下一步优化建议

1. 引入“最小竞争分数”动态阈值剪枝（early skip）。
2. 评估 WAND / Block-Max WAND 以减少热点词全量打分。
3. 对布尔查询路径引入分块并行 + 局部 Top-K 合并。
4. 建立固定压测基线（固定数据、固定 CPU、固定 `-count`），并用 `benchstat` 管理回归。

---

## 8. 风险与注意事项

1. 快路径与慢路径必须保持评分语义一致。
2. 同分排序规则变更可能影响旧测试的顺序断言。
3. 基准数据需固定生成策略，避免“数据漂移”导致误判优化效果。
