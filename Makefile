.PHONY: all build test lint bench clean help fmt run run-cli demo install

# Maure 搜索引擎构建脚本

help: ## 显示帮助
	@echo "Maure 搜索引擎"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## 构建 CLI 工具
	go build -ldflags="-s -w" -o bin/maure ./cmd/maure

install: build ## 构建并安装到 PATH
	cp bin/maure /usr/local/bin/maure || cp bin/maure $$(go env GOPATH)/bin/maure

run: ## 运行 HTTP 服务
	./bin/maure serve

run-cli: build ## 运行 CLI (默认 help)
	./bin/maure help

test: ## 运行单元测试
	go test -v ./pkg/...

demo: ## 运行集成测试
	go test -v ./test/...

lint: ## 代码检查
	go vet ./pkg/... ./cmd/...

bench: ## 基准测试
	go test -bench=. -benchmem ./pkg/...

clean: ## 清理构建产物
	rm -rf bin/
	rm -f *.prof

fmt: ## 格式化代码
	go fmt ./pkg/... ./cmd/...
