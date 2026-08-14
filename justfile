default: verify

fmt:
    gofmt -w ./cmd ./internal ./skills

fmt-check:
    test -z "$(gofmt -l ./cmd ./internal ./skills)"

vet:
    go vet ./...

test:
    go test -race ./...

build:
    mkdir -p bin
    go build -o bin/shipproof ./cmd/shipproof

verify: fmt-check vet test build
