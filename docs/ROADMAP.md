# Дорожная карта

Владелец: Platform Team
Проверено: 2026-03-25

## Done

### RM-001 Agent Harness Bootstrap
- закреплены `AGENTS.md`, карта документации, `Makefile`, локальный `make ci` и workflow для `exec-plan`

### RM-002 Go Refactor And Test Coverage
- `main.go` оставлен тонким bootstrap
- `config`, `audio`, `providers`, `outputs` и оркестрация вынесены в отдельные пакеты
- добавлен базовый `unit coverage` на критичных слоях

### RM-003 12-Factor Cleanup
- основной `runtime contract` сделан `env-first`
- локальные `runtime outputs` и граница `config/env` вынесены из `git-tracked workflow`

### RM-004 OpenAI Capability Registry
- формализован `model capability gating`
- сохранён `path` для `gpt-4o-transcribe-diarize`

### RM-005 Groq Support
- добавлена поддержка `whisper-large-v3` и `whisper-large-v3-turbo`

### RM-006 Subtitle Options
- `timestamps`, `srt`, `vtt` включаются только для `capability-compatible models`

### RM-007 Diarization
- добавлен `provider-native diarization path` через OpenAI

### RM-009 Auto-Convert Common Audio And Video Formats To M4A
- поддерживаемые non-`m4a` аудио- и видеоформаты автоматически нормализуются в `m4a` через явный `preprocessing step`
- промежуточные `source/chunk` артефакты вынесены в `<output>/<base>/_work/`, а `provider adapters` остались без скрытой orchestration-логики

## In Progress

- нет

## Planned
- нет

## Blocked

### RM-008 OpenRouter Follow-Up
- заблокировано до подтверждённого `official transcription contract`
