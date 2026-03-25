# Roadmap

Owner: Platform Team
Last Verified: 2026-03-25

## Done

### RM-001 Agent Harness Bootstrap
- закреплены `AGENTS.md`, docs map, `Makefile`, локальный `make ci` и exec-plan workflow

### RM-002 Go Refactor And Test Coverage
- `main.go` оставлен тонким bootstrap
- config, audio, providers, outputs и orchestration вынесены в отдельные пакеты
- добавлен baseline unit coverage на критичных слоях

### RM-003 12-Factor Cleanup
- основной runtime contract сделан env-first
- локальные runtime outputs и config/env boundary вынесены из git-tracked workflow

### RM-004 OpenAI Capability Registry
- formalized model capability gating
- сохранён path для `gpt-4o-transcribe-diarize`

### RM-005 Groq Support
- добавлена поддержка `whisper-large-v3` и `whisper-large-v3-turbo`

### RM-006 Subtitle Options
- `timestamps`, `srt`, `vtt` включаются только для capability-compatible models

### RM-007 Diarization
- добавлен provider-native diarization path через OpenAI

## In Progress

- none

## Planned

### RM-009 Auto-Convert Common Audio And Video Formats To M4A
- автоматически конвертировать распространённые аудио- и видеоформаты в `m4a` через `ffmpeg` перед транскрипцией
- сохранить это как явный preprocessing step, а не как скрытую побочную магию внутри provider adapters

## Blocked

### RM-008 OpenRouter Follow-Up
- blocked до подтверждённого official transcription contract
