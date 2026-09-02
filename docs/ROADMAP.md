# Дорожная карта

Владелец: Platform Team
Проверено: 2026-09-02

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
- после RM-014 registry содержит только актуальный `gpt-transcribe`

### RM-005 Groq Support
- добавлена поддержка `whisper-large-v3` и `whisper-large-v3-turbo`

### RM-006 Subtitle Options
- `timestamps`, `srt`, `vtt` включаются только для `capability-compatible models`

### RM-007 Diarization
- исторически был добавлен `provider-native diarization path` через OpenAI
- в RM-014 этот path полностью удалён вместе с public artifact и live target ради единственного OpenAI default

### RM-009 Auto-Convert Common Audio And Video Formats To M4A
- поддерживаемые non-`m4a` аудио- и видеоформаты автоматически нормализуются в `m4a` через явный `preprocessing step`
- введён явный work layout для `source/chunk`; в RM-013 он перенесён из public output tree в системный cache

### RM-011 Bash Completion
- добавлены `whisper-cli completion bash` и `make install-bash-completion` для предсказуемого Linux install contract
- `provider`, `outputs` и совместимые `models` теперь подсказываются детерминированно без запуска runtime/preflight path

### TD-009 Cobra CLI Runtime Contract Cleanup
- `help`, `subcommands` и `bash completion` переведены на `cobra`
- runtime contract сокращён до `flags > env > defaults`, а `legacy YAML config` удалён
- публичный CLI использует GNU long flags вроде `--input` и `--provider`

### RM-010 Secure OpenAI API Key Resolution
- OpenAI API key в Linux runtime резолвится через Secret Service (`secret-tool lookup application whisper-cli provider openai kind api-key`)
- `OPENAI_API_KEY` сохранён как fallback для CI и opt-in live smoke tests
- `app.NewDefault()` больше не читает OpenAI secret, а `--help` и `completion bash` остаются без secret lookup, `ffmpeg` и provider preflight

### RM-013 Cache-First Multi-Input Output
- `whisper-cli --input *.m4a` принимает shell-expanded ordered batch и обрабатывает files последовательно
- default public output сокращён до `<source-stem>.txt`, а canonical JSON и work artifacts перенесены в системный cache
- public output collisions проверяются до transcription requests

### RM-014 GPT Transcribe As The Only OpenAI Model
- `gpt-transcribe` стал единственной поддерживаемой и default-моделью OpenAI
- `--language` сериализуется для OpenAI как один `languages[]`, а singular API field не отправляется
- старые OpenAI model IDs отклоняются явно без silent migration
- diarization-контур удалён; Groq models и generic provider/model abstraction сохранены

## In Progress

- нет

## Planned

### RM-012 Secure Groq API Key Resolution
- определить безопасный runtime contract для `GROQ_API_KEY`
- решить, должен ли Groq повторить Secret Service priority order или получить отдельный provider-specific contract

## Blocked

### RM-008 OpenRouter Follow-Up
- заблокировано до подтверждённого `official transcription contract`
