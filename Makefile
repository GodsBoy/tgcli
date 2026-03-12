VERSION ?= 0.1.0
BINARY  := tgcli
CGO_FLAGS := CGO_ENABLED=1
BUILD_TAGS := -tags sqlite_fts5
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install test lint vet clean

build:
	$(CGO_FLAGS) go build $(BUILD_TAGS) $(LDFLAGS) -o dist/$(BINARY) ./cmd/tgcli

install:
	$(CGO_FLAGS) go install $(BUILD_TAGS) $(LDFLAGS) ./cmd/tgcli

test:
	$(CGO_FLAGS) go test $(BUILD_TAGS) -race -count=1 ./...

vet:
	go vet $(BUILD_TAGS) ./...

lint:
	golangci-lint run $(BUILD_TAGS) ./...

clean:
	rm -rf dist/
