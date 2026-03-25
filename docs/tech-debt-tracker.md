# Tech Debt Tracker

Owner: Platform Team
Last Verified: 2026-03-25

## Active

### TD-001 OpenRouter Transcription Contract Is Unclear
- Impact: нельзя безопасно объявить OpenRouter как supported transcription provider без drift относительно official API surface.
- Plan: при возврате к slice `RM-008` повторно проверить official docs/OpenAPI и либо реализовать adapter, либо явно оставить provider blocked.

### TD-004 Policy For Incompatible Optional Artifacts Is Still Partial
- Impact: CLI теперь автоматически отбрасывает `timestamps` для incompatible models, но policy для `srt/vtt` остаётся strict-error и не централизована как единый UX contract.
- Plan: определить, какие optional artifacts должны auto-downgrade, какие должны оставаться explicit errors, и зафиксировать это в одном capability negotiation layer.

### TD-005 Missing Binary-Level CLI Smoke Coverage
- Impact: регрессии вроде fatal на `-h/--help` ловятся только ручным запуском, потому что current tests в основном package-level и не проверяют entrypoint UX/end-to-end exit behavior.
- Plan: добавить лёгкие smoke tests для help, parse failures и базового happy path бинарника без live-provider dependency.

### TD-006 Batch Output Directories Collide On Shared Basenames
- Impact: при batch run файлы вроде `lecture.m4a` и `lecture.mp3` пишут артефакты в один и тот же `<output-dir>/lecture`, что создаёт риск silent overwrite, смешивания chunk artifacts и потери данных.
- Plan: сделать output naming collision-safe для directory input (например, сохранять relative path или suffix с extension) и добавить directory-level regression test.

### TD-007 Chunk Transcription Does Not Fail Fast On First Error
- Impact: после первой ошибки уже queued или in-flight chunks продолжают транскрибироваться, из-за чего пользователь дольше ждёт ошибку и платит за лишние provider requests.
- Plan: перевести chunk orchestration на `context.WithCancel`, прекращать scheduling после первой ошибки и добавить tests на cancel/error path.

### TD-008 Provider Contract Coverage Is Thin Outside OpenAI
- Impact: `internal/provider` helper layer остаётся без unit coverage, а Groq adapter покрыт в основном только preflight test, поэтому drift в retry/parser/request-shape может пройти через `make ci`.
- Plan: добавить deterministic unit tests для `Retry`, `ParseOpenAICompatibleTranscript`, `MarshalRawArray` и Groq request construction/response parsing.
## Closed

### TD-003 добавить make targets build/install
- Resolved: добавлены `make build` и `make install`; `install` по умолчанию копирует `./bin/whisper-cli` в `~/.local/bin/whisper-cli`, а contract зафиксирован в `README.md` и `docs/MAINTENANCE.md`.

### TD-002 настроить линтер
  - Resolved: добавлен локальный `golangci-lint` v2 baseline, `make lint` и включение lint в `make ci`
