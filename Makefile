APP_NAME := gitdex
CMD_PATH := ./cmd/gitdex
BIN_DIR  := bin
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GOOS     ?= $(shell go env GOOS)
GOARCH   ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
  EXT := .exe
else
  EXT :=
endif

DEV_BIN     := $(BIN_DIR)/$(APP_NAME)$(EXT)
RELEASE_BIN := $(BIN_DIR)/$(APP_NAME)-$(GOOS)-$(GOARCH)$(EXT)

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build release release-assets test race clean fmt lint cutover-preflight

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(DEV_BIN) $(CMD_PATH)
	@echo "Built: $(DEV_BIN)"

release:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(RELEASE_BIN) $(CMD_PATH)
	@echo "Built: $(RELEASE_BIN)"

release-assets:
	./scripts/build.sh $(VERSION) dist

test:
	go test ./...

race:
	go test -race ./...

clean:
	rm -rf $(BIN_DIR)

fmt:
	gofmt -w .

lint:
	go vet ./...

cutover-preflight:
	./scripts/v3-cutover-preflight.sh
