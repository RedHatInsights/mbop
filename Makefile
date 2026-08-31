all: build

build:
	go build cmd/mbop/mbop.go

run:
	go run cmd/mbop/mbop.go

clean:
	rm -f mbop
	go clean -cache

test:
	go test ./...

lint:
	golangci-lint run --enable=errcheck,gocritic,gofmt,goimports,gosec,gosimple,govet,ineffassign,revive,staticcheck,typecheck,unused,bodyclose --fix=false --max-same-issues=20  --print-issued-lines=true --print-linter-name=true --sort-results=true

fix:
	golangci-lint run --enable=errcheck,gocritic,gofmt,goimports,gosec,gosimple,govet,ineffassign,revive,staticcheck,typecheck,unused,bodyclose --fix=true --max-same-issues=20  --print-issued-lines=true --print-linter-name=true --sort-results=true


.PHONY: build clean lint fix test run
