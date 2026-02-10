# CLI / API 参考

## CLI

### 常用命令

```bash
# 初始化索引
maure init /tmp/maure-demo/index

# 导入日志
maure --index /tmp/maure-demo/index parse-log /tmp/maure-demo/app.log --format=auto

# 搜索
maure --index /tmp/maure-demo/index search "error OR timeout"
```

### 高级查询语法

#### 1) 范围查询（字段级）

```bash
maure search "price:[100 TO 300]"
maure search "timestamp:[2026-02-10T09:00:00Z TO 2026-02-10T10:00:00Z]"
```

- 支持类型：数值、时间
- 时间推荐 RFC3339

#### 2) 通配符查询（字段级）

```bash
maure search "title:iph*"
```

- 仅支持后缀 `*`（prefix 查询）
- 不支持前导 `*`
- 不支持 `?`

#### 3) 模糊查询（字段级）

```bash
maure search "name:roam~1"
```

- 仅支持 `~1`
- 不支持 `~2` 及以上

#### 4) 布尔组合

```bash
maure search "price:[100 TO 300] AND title:iph*"
maure search "price:[100 TO 500] NOT title:iph*"
```

## HTTP API

### 搜索

```bash
curl "http://127.0.0.1:8080/search?q=price:[100 TO 300] AND title:iph*"
```

说明：复杂查询建议 URL 编码。

### 统计

```bash
curl "http://127.0.0.1:8080/stats"
```

### 文档详情

```bash
curl "http://127.0.0.1:8080/doc/1"
```
