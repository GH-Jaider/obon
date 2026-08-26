BINARY := obon
VERSION ?= $(shell date +%Y.%m.%d)
GOFILES := $(shell find . -name '*.go')
STATICCHECK := $(shell go env GOPATH)/bin/staticcheck

.PHONY: build test vet check fmt install run clean release

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/obon

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet
	@if [ -x "$(STATICCHECK)" ]; then $(STATICCHECK) ./...; else echo "staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)"; fi

install:
	go install ./cmd/obon

run: build
	./bin/$(BINARY)

clean:
	rm -rf bin dist

release: check test
	goreleaser release --snapshot --clean
