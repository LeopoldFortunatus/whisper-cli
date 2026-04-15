# Дорожная карта

Владелец: Platform Team
Проверено: 2026-04-14

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

### RM-011 Bash Completion
- добавлены `whisper-cli completion bash` и `make install-bash-completion` для предсказуемого Linux install contract
- `provider`, `outputs` и совместимые `models` теперь подсказываются детерминированно без запуска runtime/preflight path

### TD-009 Cobra CLI Runtime Contract Cleanup
- `help`, `subcommands` и `bash completion` переведены на `cobra`
- runtime contract сокращён до `flags > env > defaults`, а `legacy YAML config` удалён
- публичный CLI использует GNU long flags вроде `--input` и `--provider`

## In Progress

- нет

## Planned

### RM-010 Secure Linux API Key Resolution
- определить безопасный `runtime contract` для чтения API keys в Linux, включая допустимые источники секретов и их приоритет
- решить, как и откуда CLI должен получать ключи без утечки в shell history, `process list`, world-readable config files или `git-tracked state`

## Blocked

### RM-008 OpenRouter Follow-Up
- заблокировано до подтверждённого `official transcription contract`
