# Maure 测试策略

## 测试原则

1. **核心功能优先**：测试最重要的功能
2. **集成测试为主**：端到端验证
3. **不过度测试**：50% 覆盖率即可
4. **性能测试简单**：基准测试记录关键指标

## 测试类型

### 单元测试

对独立函数/方法进行测试。

```go
func TestTokenizer(t *testing.T) {
    tokenizer := NewStandardTokenizer()
    // 测试分词结果
}
```

### 集成测试

验证多个组件协同工作。

```go
func TestIndexWorkflow(t *testing.T) {
    // 1. 创建索引
    // 2. 添加文档
    // 3. 搜索验证
}
```

### 基准测试

记录性能指标。

```go
func BenchmarkIndexAdd(b *testing.B) {
    // 测量添加文档性能
}
```

## 运行测试

```bash
make test        # 运行所有测试
make bench       # 运行基准测试
go test -v ./pkg/document    # 测试单个包
go test -run TestXXX         # 运行特定测试
go test -bench=.             # 只运行基准测试
```

## 测试覆盖

目标覆盖率：> 50%

```bash
go test -cover ./pkg/...
```

## 注意事项

1. 测试失败时，提供清晰的错误信息
2. 使用表驱动测试减少重复代码
3. 基准测试需要有实际意义的数据量
