# 日志检索前端示例

这个页面演示如何通过 HTTP API 使用 Maure 完成日志检索任务：

- 错误检索（`error OR failed OR timeout`）
- 分组聚合（`group=level` / `group=time(5m)`）
- 命中详情查看（`/doc/:id`）

## 1. 准备日志并写入索引

```bash
rm -rf /tmp/maure-ui-demo
mkdir -p /tmp/maure-ui-demo

cat > /tmp/maure-ui-demo/app.log <<'LOG'
{"timestamp":"2026-02-10T09:00:00Z","level":"error","message":"request failed: db timeout"}
{"timestamp":"2026-02-10T09:01:00Z","level":"warn","message":"retrying payment task"}
{"timestamp":"2026-02-10T09:02:00Z","level":"info","message":"job started"}
{"timestamp":"2026-02-10T09:03:00Z","level":"error","message":"connection exception"}
LOG

GOCACHE="/Users/kayz/Projects/Go/Maure/.cache/go-build" go run ./cmd/maure init /tmp/maure-ui-demo/index
GOCACHE="/Users/kayz/Projects/Go/Maure/.cache/go-build" go run ./cmd/maure --index /tmp/maure-ui-demo/index parse-log --log-format=json /tmp/maure-ui-demo/app.log
```

## 2. 启动搜索 API

```bash
GOCACHE="/Users/kayz/Projects/Go/Maure/.cache/go-build" go run ./cmd/maure --index /tmp/maure-ui-demo/index serve --port 8080
```

## 3. 启动前端页面

在另一个终端执行：

```bash
python3 -m http.server 5173 -d examples/frontend/log-console
```

浏览器打开：

- http://127.0.0.1:5173

页面默认 API 地址是 `http://127.0.0.1:8080`。

## 4. 你应看到的效果

- 顶部显示索引统计：文档数、词项数
- 点击“执行排障任务”后显示错误相关命中
- 右侧出现 `level` 聚合柱状图
- 每个命中可看到 `DocID/Score/Highlight` 与文档字段摘要
