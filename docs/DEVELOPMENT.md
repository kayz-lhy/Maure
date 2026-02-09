# Maure 开发指南

## 环境要求

- Go 1.21+
- git

## 开始开发

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/maure.git
cd maure
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 构建项目

```bash
make build
```

### 4. 运行测试

```bash
make test
```

## 项目结构

```
maure/
├── cmd/maure/          # CLI 入口
├── pkg/                # 核心代码
│   ├── analyzer/       # 分词器
│   ├── document/       # 文档
│   ├── index/          # 索引
│   ├── query/          # 查询
│   ├── search/         # 评分
│   └── store/          # 存储
├── docs/               # 文档
└── Makefile            # 构建脚本
```

## 开发命令

| 命令 | 说明 |
|------|------|
| `make build` | 构建 |
| `make test` | 测试 |
| `make bench` | 基准测试 |
| `make lint` | 代码检查 |
| `make fmt` | 格式化 |
| `make clean` | 清理 |

## 代码风格

- 使用 `gofmt` 格式化代码
- 公开 API 必须有注释
- 关键逻辑需要注释
- 错误处理要明确

## 添加新功能

1. 在对应 pkg 目录下添加代码
2. 编写测试（至少集成测试）
3. 运行测试确保通过
4. 更新文档（如需要）

## 提交代码

```bash
# 创建分支
git checkout -b feature/xxx

# 开发、测试

# 提交
git add .
git commit -m "feat: xxx"

# 推送并创建 PR
```
