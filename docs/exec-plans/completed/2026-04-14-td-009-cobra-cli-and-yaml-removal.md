# TD-009 Cobra CLI и удаление YAML runtime contract

Владелец: Platform Team
Проверено: 2026-04-14
Status: Completed

## Цель

Перевести CLI на `cobra`, убрать hand-rolled `help/completion`, удалить `YAML` runtime contract и зафиксировать новый пользовательский контракт `flags > env > defaults`.

## Контекст

- Текущий CLI использует собственный `flag.FlagSet`, собственный `bash` script generator и поддерживает `config.yaml` как legacy runtime source.
- `TD-009` делает сознательный breaking change: long flags переходят на GNU-форму `--input`, `--provider` и т.д.; синтаксис `-input` больше не поддерживается.
- Completion должен продолжать работать без `API key`, `ffmpeg` и normal transcription flow.

## Риски

1. Случайно смешать CLI migration с изменением runtime orchestration и сломать рабочий transcription path.
2. Потерять completion-подсказки для `provider/model/outputs` при переходе с hand-written bash logic на `cobra`.
3. Оставить docs drift, если удалить `YAML` только из кода, но не из `README`/архитектурных документов.
4. Сломать `vendor`-режим, если зависимость на `cobra` будет добавлена без обновления `vendor/`.

## План

1. Упростить `internal/config` до typed overrides и `Resolve(flags, env)` без `YAML` и filesystem-зависимостей.
2. Перевести `internal/app` на входной `config.Config`, чтобы runtime больше не знал про CLI parsing.
3. Добавить `internal/cli` с `cobra` root command, `completion bash` и custom completion functions.
4. Удалить `internal/completion`, legacy `YAML` код, `config.example.yaml` и старые тесты; добавить новые command/config/completion tests.
5. Обновить `go.mod`, `go.sum`, `vendor/`, docs и docs-check gate, затем закрыть план.

## Проверка

- `GOFLAGS=-mod=vendor go test ./...`
- `make ci`
- `make install-bash-completion BASH_COMPLETION_DIR=/tmp/whisper-cli-completion`

## Журнал решений

- 2026-04-14: `YAML` удаляется сразу, без transitional compatibility layer.
- 2026-04-14: CLI переходит на GNU long flags без shim для `-input`-style синтаксиса.

## Находки

- `go test ./...` на baseline зелёный до начала миграции.
- В локальном `GOMODCACHE` уже есть `github.com/spf13/cobra` и `github.com/spf13/pflag`, поэтому `vendor` можно обновить без нового design decision по версии.
- `cobra`-generated `bash` script требует `_get_comp_words_by_ref` в smoke-окружении; для интеграционного теста достаточно минимального helper stub без зависимости от системного `bash-completion`.
- `scripts/docs_check.sh` содержал устаревшее требование на `config.example.yaml`; gate пришлось синхронизировать с новым runtime contract.

## Следующие шаги

- Отдельно решить, нужен ли compatibility window или release note для breaking change с переходом на GNU long flags.

## Итоги

- CLI переведён на `cobra`: `help`, `completion bash`, unknown-flag handling и dynamic completion теперь живут в `internal/cli`.
- Runtime contract сокращён до `flags > env > defaults`; `config.yaml`, `--config`, `input_file`, `usergpt4` и `config.example.yaml` удалены.
- `internal/app` больше не парсит `args` и принимает уже готовый `config.Config`.
- Добавлены `command tests` для `--help`, `completion bash`, unknown flags, missing input, удалённого `--config` и отказа от `-input`.
- Добавлен `bash completion smoke test` через реальный бинарник и сгенерированный script; проверены `provider/model/outputs` completions без `API key` и `ffmpeg`.
- Синхронизированы `README.md`, `AGENTS.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/ROADMAP.md`, `docs/tech-debt-tracker.md` и `scripts/docs_check.sh`.
