.DEFAULT_GOAL := build

.PHONY: clean
clean:
	rm -rf dist
	rm -rf frontend/dist

dist:
	mkdir dist

dist/apiLambda.zip: dist $(shell find . -iname "*.go")
	./scripts/build-lambda.sh github.com/jonsabados/saturdaysspinout/cmd/lambda-based-api dist/apiLambda.zip

build: dist/apiLambda.zip

frontend/dist: $(shell find frontend/src -type f) frontend/package.json frontend/package-lock.json frontend/index.html
	cd frontend && npm ci && VITE_API_BASE_URL=$$(terraform -chdir=../terraform output -raw api_url) npm run build

.PHONY: build-frontend
build-frontend: frontend/dist

.PHONY: deploy-frontend
deploy-frontend: frontend/dist
	aws s3 sync frontend/dist s3://$$(terraform -chdir=terraform output -raw frontend_bucket_name) --delete --cache-control "max-age=31536000" --exclude "index.html"
	aws s3 cp frontend/dist/index.html s3://$$(terraform -chdir=terraform output -raw frontend_bucket_name)/index.html --cache-control "max-age=300"

.PHONY: deploy-website
deploy-website:
	aws s3 sync website s3://$$(terraform -chdir=terraform output -raw website_bucket_name) --delete --cache-control "max-age=300"

.PHONY: run-rest-api
run-rest-api:
	env $$(terraform -chdir=terraform output -raw app_env_vars) LOG_LEVEL=trace go run github.com/jonsabados/saturdaysspinout/cmd/standalone-api

.PHONY: run-frontend
run-frontend:
	cd frontend && npm install && npm run dev

.PHONY: test
test:
	go test ./...

.PHONY: frontend-test
frontend-test:
	cd frontend && npm install && npm run test:run

.PHONY: ci-test
ci-test:
	go install github.com/jstemmer/go-junit-report/v2@latest
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | $$(go env GOPATH)/bin/go-junit-report -set-exit-code > test-report.xml

# Local DynamoDB for testing
DYNAMO_CONTAINER_NAME := saturdays-racelog-dynamodb
DYNAMO_PORT := 8000

.PHONY: dynamo-start
dynamo-start:
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(DYNAMO_CONTAINER_NAME)$$'; then \
		echo "Starting existing container..."; \
		docker start $(DYNAMO_CONTAINER_NAME); \
	else \
		echo "Creating new DynamoDB container..."; \
		docker run -d --name $(DYNAMO_CONTAINER_NAME) -p $(DYNAMO_PORT):8000 amazon/dynamodb-local; \
	fi
	@echo "DynamoDB local running on http://localhost:$(DYNAMO_PORT)"

.PHONY: dynamo-stop
dynamo-stop:
	@docker stop $(DYNAMO_CONTAINER_NAME) 2>/dev/null || echo "Container not running"

.PHONY: dynamo-rm
dynamo-rm: dynamo-stop
	@docker rm $(DYNAMO_CONTAINER_NAME) 2>/dev/null || echo "Container not found"

.PHONY: dynamo-status
dynamo-status:
	@docker ps -a --filter "name=$(DYNAMO_CONTAINER_NAME)" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

.PHONY: generate-mocks
generate-mocks:
	docker run --rm -v "$(PWD)://src" -w //src vektra/mockery:3