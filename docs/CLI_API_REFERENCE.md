# CLI / API 参考

## CLI

### 搜索

```bash
maure search [--from <offset>] [--size <limit>] "<query>"
```

- `--from`：分页起始偏移，默认 `0`
- `--size`：分页大小，默认 `20`，最大 `200`
- `-n`：旧参数，等价于 `--size`（已弃用）

示例：

```bash
maure search --from 0 --size 20 "golang"
maure search --from 20 --size 20 "golang"
```

## HTTP API

### 搜索接口

```http
GET /search?q=<query>&from=<offset>&size=<limit>
```

参数：

- `q`：查询语句（必填）
- `from`：分页起始偏移，默认 `0`
- `size`：分页大小，默认 `20`，最大 `200`

返回示例：

```json
{
  "total": 20,
  "total_returned": 20,
  "from": 0,
  "size": 20,
  "results": []
}
```
