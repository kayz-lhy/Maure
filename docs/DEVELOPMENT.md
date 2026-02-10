# Maure 开发指南

## 环境要求

- Go `1.24+`
- git

## 初始化

```bash
git clone <your-repo-url>
cd maure
go mod download
```

## 常用命令

```bash
make build      # 构建 bin/maure
make test       # pkg 单测
make demo       # test/ 集成测试
make lint       # go vet
make fmt        # go fmt
```

补充建议（提交前执行）：

```bash
go test ./...
go test -race ./...
```

## 分支建议

- 功能分支统一使用前缀：`codex/`
- 示例：`codex/cli-cobra-refactor`、`codex/frontend-log-demo`

## 开发流程

1. 从基线分支切新分支。
2. 先补测试或明确验收路径，再改代码。
3. 每次修改后运行最小必要测试，再跑全量测试。
4. 同步更新文档（README / docs）。

## CLI 与 API 联调

### 启动服务

```bash
./bin/maure --index /tmp/maure-demo/index serve --port 8080
```

### 常见调试请求

```bash
curl "http://127.0.0.1:8080/stats"
curl "http://127.0.0.1:8080/search?q=error%20OR%20timeout&agg=count&group=level"
curl "http://127.0.0.1:8080/doc/1"
curl -X DELETE "http://127.0.0.1:8080/delete?id=1"
```

## 文档维护规则

1. 命令参数、接口路径变更时必须同步更新 `README.md`。
2. 若行为与历史不兼容，需标注兼容策略与弃用说明。
3. 性能问题分析与优化计划统一放在 `docs/reports/`。
