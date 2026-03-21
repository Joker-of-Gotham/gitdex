GO ?= go

.PHONY: test run daemon completion-powershell fmt

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/gitdex --help

daemon:
	$(GO) run ./cmd/gitdexd run

completion-powershell:
	$(GO) run ./cmd/gitdex completion powershell

fmt:
	gofmt -w ./cmd ./internal ./test
