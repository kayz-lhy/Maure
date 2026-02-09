# 贡献指南

感谢你的关注！Maure 是一个个人学习项目，欢迎参与贡献。

## 如何参与

### 报告问题

遇到问题时，请：
1. 先搜索是否已有类似问题
2. 使用 Issue 模板提交
3. 描述问题现象、复现步骤、预期结果

### 提交代码

1. Fork 项目
2. 创建特性分支：`git checkout -b feature/xxx`
3. 开发、测试
4. 提交：`git commit -m "feat: xxx"`
5. 推送并创建 PR

## 代码风格

- 遵循 Go 代码规范
- 使用 `gofmt` 格式化
- 公开 API 需要注释
- 关键逻辑需要注释

## 测试

- 核心功能需要测试
- 运行 `make test`
- 运行 `make bench` 检查性能影响

## 沟通

- Issue 用于问题报告
- Discussion 用于讨论
