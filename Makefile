.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "; section=""} \
		/^##@ / { section=substr($$0, 5); printf "\n\033[1m%s\033[0m\n", section; next } \
		/^[a-zA-Z_-]+:.*?## / { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Testing

.PHONY: test
test: ## Run Go tests
	go test ./...

.PHONY: frontend-test
frontend-test: ## Run frontend tests
	cd frontend && npm install && npm run test:run

.PHONY: ci-test
ci-test: ## Run Go tests with coverage (for CI)
	go install github.com/jstemmer/go-junit-report/v2@latest
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | $$(go env GOPATH)/bin/go-junit-report -set-exit-code > test-report.xml

DYNAMO_CONTAINER_NAME := saturdays-racelog-dynamodb
DYNAMO_PORT := 8000

.PHONY: dynamo-start
dynamo-start: ## Start local DynamoDB container
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(DYNAMO_CONTAINER_NAME)$$'; then \
		echo "Starting existing container..."; \
		docker start $(DYNAMO_CONTAINER_NAME); \
	else \
		echo "Creating new DynamoDB container..."; \
		docker run -d --name $(DYNAMO_CONTAINER_NAME) -p $(DYNAMO_PORT):8000 amazon/dynamodb-local; \
	fi
	@echo "DynamoDB local running on http://localhost:$(DYNAMO_PORT)"

.PHONY: dynamo-stop
dynamo-stop: ## Stop local DynamoDB container
	@docker stop $(DYNAMO_CONTAINER_NAME) 2>/dev/null || echo "Container not running"

.PHONY: dynamo-rm
dynamo-rm: dynamo-stop ## Stop and remove local DynamoDB container
	@docker rm $(DYNAMO_CONTAINER_NAME) 2>/dev/null || echo "Container not found"

.PHONY: dynamo-status
dynamo-status: ## Check local DynamoDB container status
	@docker ps -a --filter "name=$(DYNAMO_CONTAINER_NAME)" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

.PHONY: generate-mocks
generate-mocks: ## Generate test mocks with mockery
	docker run --rm -v "$(PWD)://src" -w //src vektra/mockery:3

##@ Building

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf dist
	rm -rf frontend/dist

# Go source files (wildcard isn't recursive, so enumerate depth levels - if packages get deeper modify this...)
GO_FILES := $(wildcard *.go) $(wildcard */*.go) $(wildcard */*/*.go) $(wildcard */*/*/*.go)

# Frontend source files
FRONTEND_FILES := $(wildcard frontend/src/*) $(wildcard frontend/src/*/*) $(wildcard frontend/src/*/*/*) $(wildcard frontend/src/*/*/*/*)

dist:
	mkdir dist

dist/apiLambda.zip: dist $(GO_FILES)
	./scripts/build-lambda.sh github.com/jonsabados/saturdaysspinout/cmd/lambda-based-api dist/apiLambda.zip

dist/websocketLambda.zip: dist $(GO_FILES)
	./scripts/build-lambda.sh github.com/jonsabados/saturdaysspinout/cmd/websocket-lambda dist/websocketLambda.zip

.PHONY: build
build: dist/apiLambda.zip dist/websocketLambda.zip ## Build all Lambda deployment packages

frontend/dist: $(FRONTEND_FILES) frontend/package.json frontend/package-lock.json frontend/index.html
	cd frontend && npm ci && VITE_API_BASE_URL=$$(terraform -chdir=../terraform output -raw api_url) VITE_WS_BASE_URL=$$(terraform -chdir=../terraform output -raw ws_url) npm run build

.PHONY: build-frontend
build-frontend: frontend/dist ## Build frontend for deployment

##@ Deployment

.PHONY: deploy-frontend
deploy-frontend: frontend/dist ## Deploy frontend to S3
	aws s3 sync frontend/dist s3://$$(terraform -chdir=terraform output -raw frontend_bucket_name) --delete --cache-control "max-age=31536000" --exclude "index.html"
	aws s3 cp frontend/dist/index.html s3://$$(terraform -chdir=terraform output -raw frontend_bucket_name)/index.html --cache-control "max-age=300"

.PHONY: deploy-website
deploy-website: ## Deploy static website to S3
	aws s3 sync website s3://$$(terraform -chdir=terraform output -raw website_bucket_name) --delete --cache-control "max-age=300"

##@ Local Development

.PHONY: run-rest-api
run-rest-api: ## Run backend API locally
	env $$(terraform -chdir=terraform output -raw app_env_vars) LOG_LEVEL=trace go run github.com/jonsabados/saturdaysspinout/cmd/standalone-api

.PHONY: run-frontend
run-frontend: ## Run frontend dev server
	cd frontend && npm install && VITE_WS_BASE_URL=$$(terraform -chdir=../terraform output -raw ws_url) npm run dev