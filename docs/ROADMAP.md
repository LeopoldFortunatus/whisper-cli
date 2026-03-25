# Roadmap

Owner: Platform Team
Last Verified: 2026-03-25

## Active / Planned

### RM-001 Agent Harness Bootstrap
- закрепить `AGENTS.md`, docs map, `Makefile`, CI, exec-plan workflow

### RM-002 Go Refactor And Test Coverage
- держать `main.go` тонким
- вынести config, audio, providers, outputs, orchestration
- довести baseline unit coverage на критичных слоях

### RM-003 12-Factor Cleanup
- env-first runtime
- cleanup git/runtime boundaries
- no repo-local operational state by default

### RM-004 OpenAI Capability Registry
- formalize model capabilities
- keep `gpt-4o-transcribe-diarize` path supported through normalized transcript model

### RM-005 Groq Support
- поддержка `whisper-large-v3` и `whisper-large-v3-turbo`

### RM-006 Subtitle Options
- `timestamps`, `srt`, `vtt` только для capability-compatible models

### RM-007 Diarization
- provider-native diarization path через OpenAI

### RM-008 OpenRouter Follow-Up
- blocked до подтверждённого official transcription contract
