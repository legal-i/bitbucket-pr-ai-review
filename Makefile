SHELL:=/bin/bash


GOLANG_IMAGE=golang:1.22.3

ifeq ($(OS),Windows_NT)
    detected_OS := Windows
else
    detected_OS := $(shell sh -c 'uname 2>/dev/null || echo Unknown')
endif

upgrade-libraries:
	@go get -u ./...
	@go fmt ./...
	@go mod tidy
	@go mod verify

gofmt:
	@docker run -w /src -v $(shell pwd):/src $(GOLANG_IMAGE) go fmt ./...

build:
ifeq ($(detected_OS),Windows)
	@docker run -w /src -v $(shell pwd):/src $(GOLANG_IMAGE) env GOOS=windows GOARCH=amd64 go build -ldflags='-s' -o review.exe main.go
endif
ifeq ($(detected_OS),Darwin) # Mac OS X
	@docker run -w /src -v $(shell pwd):/src $(GOLANG_IMAGE) env GOOS=darwin GOARCH=amd64 go build -ldflags='-s' -o review main.go
endif
ifeq ($(detected_OS),Linux)
	@docker run -w /src -v $(shell pwd):/src $(GOLANG_IMAGE) env GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o review main.go
endif
