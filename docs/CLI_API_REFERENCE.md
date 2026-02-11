# CLI / API 参考

## DSL 说明（V1）

查询语法由独立 DSL 管线处理：`contracts -> pipeline -> components -> query adapter`。  
常用示例：

```text
price:[100 TO 300] AND title:iph*
name:roam~1 OR title:iph*
field:*      # 字段存在查询
```

说明：

- `IN index("...")`、`LIMIT`、`SORT BY` 已进入 QueryPlan 元信息；
- `REQUIRE_IN` 可选约束已支持：声明后必须配合 `IN index("...")`；
- 若调用路径未消费该元信息，会返回明确错误，不会静默忽略。

## HTTP API

### 搜索接口

```http
GET /search?q=<query>&include_doc=true&fields=message,level
```

参数：

- `q`：查询语句（必填）
- `include_doc`：可选，`true/1/yes` 时返回 `doc.summary`
- `fields`：可选，逗号分隔字段白名单，示例 `fields=message,level,timestamp`

行为：

- 默认（不传 `include_doc` 且不传 `fields`）仅返回 `doc_id/score/highlights`
- 传 `include_doc=true` 返回 `doc.summary`
- 传 `fields` 返回 `doc.fields`（仅白名单字段）
- 仅传 `fields` 也会返回 `doc`，用于替代前端逐条 `/doc` 请求

返回示例：

```json
[
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
]
```
