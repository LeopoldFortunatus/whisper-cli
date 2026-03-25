GO ?= go
GOFLAGS ?= -mod=vendor
GOFILES := $(shell find . -path ./vendor -prune -o -name '*.go' -print)

.PHONY: fmt fmt-check test vet docs-check ci test-live-openai test-live-openai-diarize test-live-groq

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@test -z "$$(gofmt -l $(GOFILES))" || (echo "Run make fmt"; gofmt -l $(GOFILES); exit 1)

test:
	GOFLAGS=$(GOFLAGS) $(GO) test ./...

vet:
	GOFLAGS=$(GOFLAGS) $(GO) vet ./...

docs-check:
	bash scripts/docs_check.sh

ci: fmt-check vet test docs-check

test-live-openai:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/openaiadapter -run TestLiveOpenAITranscription -count=1

test-live-openai-diarize:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/openaiadapter -run TestLiveOpenAIDiarization -count=1

test-live-groq:
	@test -n "$(LIVE_AUDIO_FILE)" || (echo "LIVE_AUDIO_FILE is required"; exit 1)
	WHISPER_CLI_RUN_LIVE=1 WHISPER_CLI_LIVE_AUDIO="$(LIVE_AUDIO_FILE)" GOFLAGS=$(GOFLAGS) $(GO) test ./internal/provider/groqadapter -run TestLiveGroqTranscription -count=1
