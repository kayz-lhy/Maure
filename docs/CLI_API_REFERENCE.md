# CLI / API 速查

## CLI

### 全局参数

- `--index <path>` 索引目录
- `--format <text|json>` CLI 输出格式
- `--analyzer <standard|ram>` 分析器
- `-v, --verbose` 详细输出

### 常用命令

```bash
# 初始化
maure init /tmp/maure/index

# 导入日志
maure --index /tmp/maure/index parse-log --log-format=auto /tmp/maure/app.log

# 搜索 + 聚合
maure --index /tmp/maure/index search --group=level "error OR timeout"

# 查看统计
maure --index /tmp/maure/index stats

# 启动服务
maure --index /tmp/maure/index serve --port 8080
```

### 兼容写法说明

- `parse-log ... --format=json`：仍可用，但会提示弃用，建议改 `--log-format`。
- `search "q" --group=level`：仍可用，但会提示弃用，建议将 flag 放在前面。

## HTTP API

### 查询

```bash
curl "http://127.0.0.1:8080/search?q=error%20OR%20timeout&agg=count&group=level"
```

### 文档详情

```bash
curl "http://127.0.0.1:8080/doc/1"
```

### 统计

```bash
curl "http://127.0.0.1:8080/stats"
```

### 添加文档

```bash
curl -X POST "http://127.0.0.1:8080/add" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-1","fields":{"message":"request failed","level":"error"}}'
```

### 删除文档

```bash
curl -X DELETE "http://127.0.0.1:8080/delete?id=1"
```

说明：删除接口目前采用 query 参数，不是 RESTful 的 `DELETE /doc/:id`。
