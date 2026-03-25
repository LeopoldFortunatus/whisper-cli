# План выполнения

Владелец: Platform Team
Проверено: 2026-03-25
Status: Completed

## Цель

Исправить UX CLI для двух сценариев: `-h/--help` должен завершаться штатно с usage output, а `timestamps` должны автоматически отключаться для моделей без segment timestamps вместо fatal-ошибки.

## Контекст

- Затрагиваются `main.go`, `internal/app`, `internal/config`, `unit tests` и пользовательская документация.
- Сейчас default `outputs=timestamps` ломает запуск с `gpt-4o-transcribe`.
- Сейчас `flag.ErrHelp` доходит до `main.go` и логируется как `fatal`.

## Риски

1. Случайно ослабить валидацию не только для `timestamps`, но и для `srt/vtt/diarized`, где явный `error` пока полезен.
2. Потерять наблюдаемый `warning` о том, что `effective outputs` были изменены автоматически.
3. Исправить `help` только на уровне `parse`, но не на уровне `exit behavior` бинарника.

## План

1. Добавить `exec-path` для `help` без `fatal` и с выводом `usage`.
2. Нормализовать `timestamps` относительно `provider capabilities` перед транскрипцией.
3. Обновить `unit tests` на оба сценария и скорректировать документацию и техдолг.

## Проверка

- `go test ./internal/config ./internal/app ./...`
- `make ci`
- ручная проверка `go run . -h`

## Журнал решений

- 2026-03-25: автоматический `downgrade` ограничиваем только `timestamps`; остальные `optional artifacts` пока остаются `explicit validation errors`.

## Находки

- `README.md` уже документирует workaround через `-outputs none`, что подтверждает UX gap в текущем поведении.
- `flag.NewFlagSet(..., ContinueOnError)` уже генерирует `usage text`, но он теряется из-за `log.Fatal` в `main.go`.

## Следующие шаги

- Решить, должен ли CLI автоматически отбрасывать и другие `incompatible artifacts` (`srt/vtt`) или сохранять `strict mode`.
- Добавить `e2e smoke coverage` для бинарника, чтобы `help/exit-code regressions` ловились до ручного запуска.

## Итоги

- `timestamps` больше не валят запуск на моделях без `segment timestamps`: артефакт автоматически убирается из `effective outputs` с `warning`.
- `-h/--help` больше не проходят через `log.Fatal`; usage печатается штатно и бинарник выходит с кодом `0`.
- Проверено через `go test ./...`, `make ci`, `make build` и ручной запуск `./bin/whisper-cli -h`.
