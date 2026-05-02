# go-frame 开发与 CI 入口
# 任意 target 失败即终止,便于本地与 CI 复用同一组命令。

.DEFAULT_GOAL := help

GO        ?= go
PKG       ?= ./...
LINTER    ?= golangci-lint
VULNCHECK ?= govulncheck

.PHONY: help
help: ## 列出所有可用目标
	@awk 'BEGIN{FS=":.*##";printf "可用目标:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## 编译全部包
	$(GO) build $(PKG)

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKG)

.PHONY: test
test: ## 跑全部单测
	$(GO) test -count=1 $(PKG)

.PHONY: race
race: ## race detector 跑全部单测
	$(GO) test -race -count=1 $(PKG)

.PHONY: lint
lint: ## golangci-lint(需先 make tools)
	$(LINTER) run

.PHONY: vuln
vuln: ## 扫描依赖 CVE(需先 make tools)
	$(VULNCHECK) $(PKG)

.PHONY: fmt
fmt: ## 格式化全部 .go 文件
	gofmt -w $(shell find . -name '*.go' -not -path './.claude/*' -not -path './.omc/*')

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: tools
tools: ## 安装 lint/vuln 工具
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

# CI 一键入口:vet + race + lint + vuln。
# CI 跑 race 而非 test:同时获得正确性与并发安全两层保证,
# 整包 race 也能稳定在 1 分钟内完成。
.PHONY: ci
ci: vet race lint vuln ## CI 一键流水线
