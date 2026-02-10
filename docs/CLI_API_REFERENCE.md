# CLI / API 参考

## CLI

### 全局参数

- `--index <path>` 索引目录路径（默认 `.`）
- `--format <text|json>` 输出格式（默认 `text`）
- `--analyzer <standard|ram>` 分析器（默认 `standard`）
- `-v, --verbose` 详细输出

### 常用命令

```bash
# 初始化索引
maure init /tmp/maure/index

# 导入日志（自动识别格式）
maure --index /tmp/maure/index parse-log --log-format=auto /tmp/maure/app.log

# 搜索（布尔查询）
maure --index /tmp/maure/index search "level:error AND message:timeout"

# 聚合统计
maure --index /tmp/maure/index search --agg=count --group=level "error OR timeout"

# 启动服务
maure --index /tmp/maure/index serve --port 8080
```

## HTTP API

### `GET /search`

示例：

```http
GET /search?q=error%20OR%20timeout&from=0&size=20&include_doc=true&fields=message,level&agg=count&group=level
```

参数：

- `q`：查询语句（必填）
- `from`：分页起始偏移，默认 `0`
- `size`：分页大小，默认 `20`，最大 `200`
- `include_doc`：可选，`true/1/yes` 时返回 `doc.summary`
- `fields`：可选，逗号分隔字段白名单，例：`message,level,timestamp`
- `agg`：可选，当前支持 `count`
- `group`：可选，例：`level`、`time(5m)`

行为：

- 默认（不传 `include_doc` 且不传 `fields`）仅返回 `doc_id/score/highlights`
- 传 `include_doc=true` 返回 `doc.summary`
- 传 `fields` 返回 `doc.fields`（仅白名单字段）
- 传 `agg/group` 时，响应包含 `aggregations`

返回示例：

```json
{
  "total": 2,
  "total_returned": 2,
  "from": 0,
  "size": 20,
  "results": [
    {
      "doc_id": 1,
      "score": 2.45,
      "highlights": [
        {
          "field": "message",
          "start": 0,
          "end": 5,
          "fragment": "error"
        }
      ],
      "doc": {
        "summary": "request failed",
        "fields": {
          "message": "request failed",
          "level": "error"
        }
      }
    }
  ],
  "aggregations": {
    "count": 2,
    "buckets": {
      "error": 2
    }
  }
}
```

### 其他接口

- `GET /`：服务信息与端点索引
- `GET /doc/<id>`：按文档 ID 读取文档
- `GET /stats`：索引统计
- `POST /add`：新增文档
- `DELETE /delete?id=<id>`：删除文档

## 兼容与限制

- `size` 上限为 `200`，超过会返回 `400`
- `fields` 中字段名仅允许字母、数字、`_`、`-`、`.`
- CLI 历史写法仍有兼容层，但建议使用“flag 在前”的标准写法
