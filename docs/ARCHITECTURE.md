# Maure 架构说明

## 1. 分层结构

```text
CLI / HTTP API
    -> Query Parser / Command Layer
        -> RAMIndex / InvertedIndex / Similarity
            -> FSDirectory + Snapshot + WAL
                -> 文件系统
```

## 2. 关键目录

```text
cmd/maure/
  main.go                  # Cobra 入口
  cobra/                   # Cobra 命令树与兼容桥接
  command/                 # 业务命令实现（搜索、导入、服务等）

pkg/
  analyzer/                # 分词
  document/                # 文档模型
  index/                   # RAMIndex 与倒排索引
  query/                   # 查询对象与解析器
  search/                  # BM25/TF-IDF
  aggregate/               # count/group 聚合
  highlight/               # 高亮提取
  logparser/               # JSON/Logback 解析
  store/                   # FSDirectory、WAL、快照
```

## 3. 索引与持久化模型

### 3.1 RAM 索引

- `InvertedIndex` 维护：
  - `term -> Postings`
  - `DocCount`
  - `FieldLength`
- 查询执行基于 RAM 结构完成评分与排序。

### 3.2 FS 持久化

- 元数据：`manifest.json`
- 快照：`snapshot_*.dat.gz`
- 预写日志：`wal_*.log`

写入路径：

1. 文档先落 WAL。
2. `Commit` 时重建快照并写盘。
3. 更新 manifest 并截断 WAL。

读取路径：

1. 加载最新快照。
2. 重放 WAL 中未提交操作。
3. 必要时执行最小修复（例如旧快照缺少字段长度）。

## 4. 查询执行路径

1. API/CLI 接收查询串。
2. `query.QueryParser` 解析为查询树。
3. 查询树在 `RAMIndex` 上执行（Term/Boolean/Phrase）。
4. 输出命中文档评分。
5. 可选执行聚合与高亮。

## 5. 当前架构权衡

1. 优点：实现简单、可读性高、易调试。
2. 代价：复杂布尔查询与大命中集下性能会下降。
3. 已识别优化方向：见 `docs/reports/search-api-performance-analysis.md`。

## 6. 兼容策略

1. CLI 从原生 `flag` 迁移到 Cobra，保留历史写法兼容与弃用提示。
2. FS 快照采用“读旧写新”策略，避免强制离线迁移。
