.DEFAULT_GOAL := build

.PHONY: clean
clean:
	rm -rf dist

dist:
	mkdir dist

dist/apiLambda.zip: dist $(shell find . -iname "*.go")
	./scripts/build-lambda.sh github.com/jonsabados/saturdays-racelog/cmd/lambda-based-api dist/apiLambda.zip

build: dist/apiLambda.zip

.PHONY: run-rest-api
run-rest-api:
	LOG_LEVEL=trace go run github.com/jonsabados/saturdays-racelog/cmd/standalone-api