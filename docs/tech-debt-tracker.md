# Реестр техдолга

Владелец: Platform Team
Проверено: 2026-09-02

## Active

### TD-012 Локальный `make ci` зависит от непиннутого Go/golangci-lint toolchain
- Влияние: официальный локальный `quality gate` не воспроизводится из обычного окружения: ambient `go` сейчас `go1.23.3` при `go.mod` `go 1.24.2`, а при явном `GOROOT=/home/arykalin/go/go1.26.3` установленный `golangci-lint` built with `go1.24.4` паникует на `file requires newer Go version go1.26`; это скрывает реальные lint-регрессии и делает `make ci` красным по причине окружения, а не кода.
- План: закрепить repo-local способ запуска совместимых `go` и `golangci-lint` versions, добавить понятный preflight/version guard в `make lint` или docs, и обновить локальную установку `golangci-lint` так, чтобы `make ci` снова был единственным воспроизводимым gate.

### TD-011 Невалидные числовые env overrides молча заменяются defaults
- Влияние: `WHISPER_CLI_CHUNK_SECONDS=abc` или `WHISPER_CLI_CONCURRENCY=abc` не дают ошибку конфигурации, а запускают CLI с default value; это скрывает misconfiguration и может менять стоимость/время транскрипции без явного сигнала пользователю.
- План: сделать `chooseInt` error-returning path для env values, добавить unit tests на invalid env integers и сохранить текущее поведение для отсутствующих env variables.

### TD-001 Контракт транскрипции OpenRouter остаётся неясным
- Влияние: нельзя безопасно объявить OpenRouter как поддерживаемый `transcription provider` без drift относительно `official API surface`.
- План: при возврате к `slice` `RM-008` повторно проверить `official docs/OpenAPI` и либо реализовать `adapter`, либо явно оставить `provider` заблокированным.

### TD-004 Политика для несовместимых optional artifacts остаётся частичной
- Влияние: CLI теперь автоматически отбрасывает `timestamps` для `incompatible models`, но policy для `srt/vtt` остаётся `strict-error` и не централизована как единый `UX contract`.
- План: определить, какие `optional artifacts` должны `auto-downgrade`, какие должны оставаться `explicit errors`, и зафиксировать это в одном `capability negotiation layer`.

### TD-005 Не хватает binary-level smoke coverage для CLI
- Влияние: регрессии вроде `fatal` на `-h/--help` ловятся только ручным запуском, потому что `current tests` в основном `package-level` и не проверяют `entrypoint UX/end-to-end exit behavior`.
- План: добавить лёгкие `smoke tests` для `help`, `parse failures` и базового `happy path` бинарника без `live-provider dependency`.

### TD-007 Транскрипция чанков не делает fail-fast при первой ошибке
- Влияние: после первой ошибки уже `queued` или `in-flight` чанки продолжают транскрибироваться, из-за чего пользователь дольше ждёт ошибку и платит за лишние `provider requests`.
- План: перевести оркестрацию чанков на `context.WithCancel`, прекращать `scheduling` после первой ошибки и добавить `tests` на `cancel/error path`.

### TD-008 Coverage для provider contract слишком тонкий вне OpenAI
- Влияние: `internal/provider helper layer` остаётся без `unit coverage`, а `Groq adapter` покрыт в основном только `preflight test`, поэтому drift в `retry/parser/request-shape` может пройти через `make ci`.
- План: добавить `deterministic unit tests` для `Retry`, `ParseOpenAICompatibleTranscript`, `MarshalRawArray` и `Groq request construction/response parsing`.

## Closed

### TD-006 Batch output directories конфликтовали при общих basename
- Решение: batch manifest вычисляет все stem-prefixed public paths до транскрипции и отклоняет разные inputs, претендующие на один destination; cache directories разделены полным SHA-256 от absolute input path.

### TD-010 Локальный `make ci` сломан устаревшим default-model тестом
- Решение: `TestResolveUsesDefaultsWithoutEnv` и пользовательская документация проверяют актуальный default `provider=openai` и `model=gpt-transcribe`.

### TD-009 CLI и bash completion всё ещё на hand-rolled implementation, а legacy YAML расширяет runtime surface
- Решение: `command tree`, `help` и `bash completion` переведены на `cobra`, `legacy YAML config` удалён, целевой контракт зафиксирован как `flags > env > defaults`, а для `help`, `parse failures` и `completion bash` добавлены `command/integration tests`.

### TD-003 добавить make targets build/install
- Решение: добавлены `make build` и `make install`; `install` по умолчанию копирует `./bin/whisper-cli` в `~/.local/bin/whisper-cli`, а `contract` зафиксирован в `README.md` и `docs/MAINTENANCE.md`.

### TD-002 настроить линтер
- Решение: добавлен локальный `golangci-lint` v2 baseline, `make lint` и включение `lint` в `make ci`.
