APP_NAME := tunny
CMD := ./cmd/tunny

.PHONY: build run proxy provider install clean fmt test check

build:
	go build -o $(APP_NAME) $(CMD)

run:
	go run $(CMD)

proxy:
	go run $(CMD) proxy

provider:
	go run $(CMD) provider

install:
	go install $(CMD)

fmt:
	go fmt ./...

test:
	go test ./...

check: fmt test

clean:
	rm -f $(APP_NAME)
