BINARY  := bin/bk
VERSION ?= 0.1.0-dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/ClaudioLazaro/bitbucket-cli/internal/build.Version=$(VERSION) \
	-X github.com/ClaudioLazaro/bitbucket-cli/internal/build.Commit=$(COMMIT) \
	-X github.com/ClaudioLazaro/bitbucket-cli/internal/build.Date=$(DATE)

.PHONY: build test lint fmt vet install clean docs

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/bk

docs:
	go run ./script/gen-docs

test:
	go test ./...

lint:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed:"; echo "$$unformatted"; exit 1; \
	fi
	go vet ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/bk

clean:
	rm -rf bin dist
