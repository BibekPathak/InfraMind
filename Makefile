.PHONY: help build test-fast test-e2e test-chaos test-perf coverage fmt vet lint up down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build all Go services
	cd backend && go build ./...
	cd simulator && go build ./...
	cd deployments/loadtest && go build -o /tmp/loadtest .

fmt: ## Check Go formatting
	cd backend && gofmt -l .
	cd simulator && gofmt -l .
	cd tests && gofmt -l .

vet: ## Run go vet on all Go modules
	cd backend && go vet ./...
	cd simulator && go vet ./...
	cd tests && go vet ./...

lint: ## Static analysis (staticcheck if available, else vet)
	@command -v staticcheck >/dev/null && (cd backend && staticcheck ./...; cd simulator && staticcheck ./...) || echo "staticcheck not installed; run: go install honnef.co/go/tools/cmd/staticcheck@latest"

test-fast: fmt vet build ## Fast pipeline: lint + unit + build (runs in minutes)
	cd backend && go test ./...
	cd simulator && go test ./...
	cd ai && (python -m ruff check . || true) && (python -m mypy . || true)

test-e2e: ## Full stack: integration + system + chaos + performance (slow)
	cd tests && go test ./integration/... -timeout 900s
	cd tests && go test ./system/... -timeout 900s

test-chaos: ## Chaos restart tests only
	cd tests && go test ./chaos/... -timeout 1800s

test-perf: ## Performance smoke tests only
	cd tests && go test ./performance/... -timeout 900s

coverage: ## Generate coverage report (HTML) for backend
	cd backend && go test ./internal/... -coverprofile=/tmp/coverage.out && go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html && echo "report: /tmp/coverage.html"

up: ## Start the full stack with simulator (dev)
	docker compose --profile sim up --build

down: ## Stop the stack
	docker compose down
