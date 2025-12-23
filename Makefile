# Copyright (c) 2025 Michael D Henderson. All rights reserved.

.PHONY: all build clean generate run test

all: build

generate:
	templ generate

build: generate
	go build ./...

run: build
	go run ./cmd/server

test:
	go test ./...

clean:
	go clean ./...
