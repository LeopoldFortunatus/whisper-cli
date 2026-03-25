GO ?= go
GOFLAGS ?= -mod=vendor
BIN_NAME ?= whisper-cli
BUILD_DIR ?= bin
BUILD_PATH ?= $(BUILD_DIR)/$(BIN_NAME)
INSTALL_DIR ?= $(HOME)/.local/bin
INSTALL_PATH ?= $(INSTALL_DIR)/$(BIN_NAME)
GOFILES := $(shell find . -path ./vendor -prune -o -name '*.go' -print)
GOLANGCI_LINT ?= golangci-lint

.PHONY: build install fmt fmt-check lint test vet docs-check ci test-live-openai test-live-openai-diarize test-live-groq

build:
	mkdir -p "$(BUILD_DIR)"
	GOFLAGS=$(GOFLAGS) $(GO) build -o "$(BUILD_PATH)" .

install: build
	mkdir -p "$(INSTALL_DIR)"
	cp "$(BUILD_PATH)" "$(INSTALL_PATH)"

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@test -z "$$(gofmt -l $(GOFILES))" || (echo "Run make fmt"; gofmt -l $(GOFILES); exit 1)

lint:
	$(GOLANGCI_LINT) run --config .golangci.yml --modules-download-mode=vendor ./...

test:
	GOFLAGS=$(GOFLAGS) $(GO) test ./...

vet:
	GOFLAGS=$(GOFLAGS) $(GO) vet ./...

docs-check:
	bash scripts/docs_check.sh

ci: fmt-check lint vet test docs-check

test-live-openai:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/openaiadapter -run TestLiveOpenAITranscription -count=1

test-live-openai-diarize:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/openaiadapter -run TestLiveOpenAIDiarization -count=1

test-live-groq:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/groqadapter -run TestLiveGroqTranscription -count=1
