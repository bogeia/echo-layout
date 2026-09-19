.PHONY: init build test generate

init:
	go install github.com/google/wire/cmd/wire@v0.7.0

build:
	mkdir -p bin/ && go build -o ./bin/ ./...

test:
	go test ./...

generate:
	go generate ./...
	go mod tidy
