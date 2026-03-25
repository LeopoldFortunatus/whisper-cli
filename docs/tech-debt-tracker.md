# Реестр техдолга

Владелец: Platform Team
Проверено: 2026-03-25

## Active

### TD-001 Контракт транскрипции OpenRouter остаётся неясным
- Влияние: нельзя безопасно объявить OpenRouter как поддерживаемый `transcription provider` без drift относительно `official API surface`.
- План: при возврате к `slice` `RM-008` повторно проверить `official docs/OpenAPI` и либо реализовать `adapter`, либо явно оставить `provider` заблокированным.

### TD-004 Политика для несовместимых optional artifacts остаётся частичной
- Влияние: CLI теперь автоматически отбрасывает `timestamps` для `incompatible models`, но policy для `srt/vtt` остаётся `strict-error` и не централизована как единый `UX contract`.
- План: определить, какие `optional artifacts` должны `auto-downgrade`, какие должны оставаться `explicit errors`, и зафиксировать это в одном `capability negotiation layer`.

### TD-005 Не хватает binary-level smoke coverage для CLI
- Влияние: регрессии вроде `fatal` на `-h/--help` ловятся только ручным запуском, потому что `current tests` в основном `package-level` и не проверяют `entrypoint UX/end-to-end exit behavior`.
- План: добавить лёгкие `smoke tests` для `help`, `parse failures` и базового `happy path` бинарника без `live-provider dependency`.

### TD-006 Batch output directories конфликтуют при общих basename
- Влияние: при `batch run` файлы вроде `lecture.m4a` и `lecture.mp3` пишут артефакты в один и тот же `<output-dir>/lecture`, что создаёт риск `silent overwrite`, смешивания `chunk artifacts` и потери данных.
- План: сделать `output naming` `collision-safe` для `directory input` и добавить `directory-level regression test`.

### TD-007 Транскрипция чанков не делает fail-fast при первой ошибке
- Влияние: после первой ошибки уже `queued` или `in-flight` чанки продолжают транскрибироваться, из-за чего пользователь дольше ждёт ошибку и платит за лишние `provider requests`.
- План: перевести оркестрацию чанков на `context.WithCancel`, прекращать `scheduling` после первой ошибки и добавить `tests` на `cancel/error path`.

### TD-008 Coverage для provider contract слишком тонкий вне OpenAI
- Влияние: `internal/provider helper layer` остаётся без `unit coverage`, а `Groq adapter` покрыт в основном только `preflight test`, поэтому drift в `retry/parser/request-shape` может пройти через `make ci`.
- План: добавить `deterministic unit tests` для `Retry`, `ParseOpenAICompatibleTranscript`, `MarshalRawArray` и `Groq request construction/response parsing`.
## Closed

### TD-003 добавить make targets build/install
- Решение: добавлены `make build` и `make install`; `install` по умолчанию копирует `./bin/whisper-cli` в `~/.local/bin/whisper-cli`, а `contract` зафиксирован в `README.md` и `docs/MAINTENANCE.md`.

### TD-002 настроить линтер
- Решение: добавлен локальный `golangci-lint` v2 baseline, `make lint` и включение `lint` в `make ci`.
