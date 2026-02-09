.PHONY: all build test lint bench clean help

# Maure 搜索引擎构建脚本

help: ## 显示帮助
	@echo "Maure 搜索引擎"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## 构建项目
	go build -ldflags="-s -w" -o bin/maure ./cmd/maure

test: ## 运行测试
	go test -v ./pkg/...

bench: ## 基准测试
	go test -bench=. -benchmem ./pkg/...

lint: ## 代码检查
	go vet ./pkg/...

clean: ## 清理
	rm -rf bin/
	rm -f *.prof

fmt: ## 格式化代码
	go fmt ./...
