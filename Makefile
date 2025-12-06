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

.PHONY: run-rest-api
run-rest-api:
	LOG_LEVEL=trace CORS_ALLOWED_ORIGINS=http://localhost:5173 go run github.com/jonsabados/saturdaysspinout/cmd/standalone-api