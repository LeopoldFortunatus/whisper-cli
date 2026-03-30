# RM-011 Bash Completion

Владелец: Platform Team
Проверено: 2026-03-27
Status: Completed

## Цель

Добавить `bash completion` для основных CLI-флагов и предсказуемый Linux install contract без запуска normal runtime/preflight path.

## Контекст

- CLI уже имеет стабильный набор флагов, но не даёт shell-level discoverability для `provider`, `model`, `outputs` и path-like аргументов.
- `provider/model` знания уже живут в adapter layer, поэтому completion нельзя собирать из отдельного hand-written bash списка с риском drift.
- completion не должен требовать `API key`, `ffmpeg` или стартовать обычный transcription flow.

## Риски

1. `bash script` может задублировать CLI metadata и начать расходиться с реальным flag contract.
2. Completion install может жёстко привязаться к пользовательскому окружению и быть неудобным для безопасной локальной проверки.
3. `provider` suggestions могут показывать planned `openrouter` и вести пользователя в гарантированный runtime error.

## План

1. Централизовать CLI flag metadata и переиспользовать её для `flag.Parse` и bash completion generation.
2. Открыть provider model metadata через adapter clients и скрыть `openrouter` из completion, не меняя parser/runtime contract.
3. Добавить `whisper-cli completion bash`, `make install-bash-completion` и unit/smoke coverage.
4. Синхронизировать `README.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md` и `docs/ROADMAP.md`.
5. Прогнать локальные проверки и закрыть задачу в `completed/`.

## Проверка

- `go test ./...`
- `make install-bash-completion BASH_COMPLETION_DIR=/tmp/whisper-cli-completion`
- bash smoke check через `source /tmp/whisper-cli-completion/whisper-cli` и вызов `_whisper_cli`
- `make ci`

## Журнал решений

- 2026-03-27: выбрать публичный surface `whisper-cli completion bash`, а не static checked-in script, чтобы completion генерировался из текущего бинарника и не дрейфовал.
- 2026-03-27: скрывать `openrouter` в completion, хотя parser всё ещё принимает этот provider как planned slot.
- 2026-03-27: использовать `~/.local/share/bash-completion/completions` как дефолтный install path и поддержать `BASH_COMPLETION_DIR` override для безопасной валидации.

## Находки

- Текущий `flag.FlagSet` help не показывал subcommand-ы, поэтому help output нужно было явно дополнить discoverability-строкой про `completion bash`.
- Для `-outputs` полезнее поддержать comma-separated completion без дубликатов, чем ограничиться плоским `compgen -W`.

## Следующие шаги

- Если появится запрос на `zsh` или `fish`, их стоит строить поверх того же metadata source, а не заводить отдельные hand-written shell scripts.

## Итоги

- Добавлены генерация и установка `bash completion` без запуска transcription runtime path.
- `provider/model/output` suggestions теперь собираются из реального CLI/provider metadata, а не из дублируемого bash списка.
- Docs и roadmap синхронизированы с новым completion/install contract.
