APP_NAME := tunny
CMD := ./cmd/tunny
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build run proxy provider install clean fmt test check

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) $(CMD)

run:
	go run $(CMD)

proxy:
	go run $(CMD) proxy

provider:
	go run $(CMD) provider

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

fmt:
	go fmt ./...

test:
	go test ./...

check: fmt test

clean:
	rm -f $(APP_NAME)
