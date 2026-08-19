# План выполнения

Владелец: Platform Team
Проверено: 2026-08-19
Status: Completed

## Цель

Сделать default output cache-first: служебные файлы хранить в системном cache, наружу писать только `<media-stem>.txt`, а один запуск `whisper-cli --input *.m4a` должен последовательно обработать все переданные файлы.

## Контекст

- Затрагиваются CLI/config contract, batch orchestration, output layout, platform filesystem wrapper, tests и пользовательская документация.
- `--input` сохраняется; shell передаёт первый expanded path как значение flag, остальные paths как positional arguments.
- Default public directory — каталог source-файла; явный `--output-dir` собирает stem-prefixed artifacts в одном каталоге.
- Provider adapters и параллельная обработка chunks внутри одного файла не меняются.

## Риски

1. Разные inputs могут вычислить один public path и привести к silent overwrite.
2. Стабильный cache directory может подмешать stale chunks от предыдущего запуска.
3. Изменение default `outputs` может случайно сохранить прежнюю auto-publication diarized/timestamp artifacts.
4. Multi-file parsing не должен сломать single input, env fallback, help и completion paths.

## План

1. Перевести config на ordered input list и принять дополнительные positional paths после `--input`.
2. Построить collision-safe batch manifest до provider requests и обрабатывать files последовательно.
3. Перенести work files и canonical `transcript.json` в `os.UserCacheDir()/whisper-cli/<input-hash>/` с очисткой точного job directory перед повторным запуском.
4. Писать plain и явно запрошенные artifacts в stem-prefixed public paths.
5. Обновить unit/regression coverage и документацию, закрыть TD-006.

## Проверка

- `go test ./internal/config ./internal/cli ./internal/output ./internal/app ./internal/audio`
- `make fmt-check`
- `make test`
- `make vet`
- `make docs-check`
- `make ci`
- live provider tests: not run, provider contract не меняется

## Журнал решений

- 2026-08-19: основной multi-file syntax — `whisper-cli --input *.m4a`; wildcard раскрывает shell, внутренний globbing не добавляется.
- 2026-08-19: default public output — `<stem>.txt` рядом с source; `--output-dir` остаётся явным override.
- 2026-08-19: cache хранит последний запуск для абсолютного input path и не используется как transcription memoization.
- 2026-08-19: per-file failures не останавливают batch; global output collisions останавливают запуск до транскрипции.
- 2026-08-19: существующие public artifacts перезаписываются, а невыбранные artifacts автоматически не удаляются.

## Находки

- Baseline `make ci` падает в `golangci-lint`: linter собран Go 1.24 и не читает packages, собранные Go 1.26; проблема уже зафиксирована как TD-012.
- Правило `.gitignore` `output/` скрывало пакет `internal/output`; оно сужено до root runtime directory `/output/`, чтобы writer и его tests были частью tracked source tree.

## Следующие шаги

- Исправление toolchain/linter compatibility остаётся в TD-012 и не входит в этот slice.

## Итоги

- `--input` принимает shell-expanded ordered batch; files обрабатываются последовательно, per-file failure не блокирует следующие inputs.
- Default public layout сокращён до `<source-stem>.txt`, canonical JSON и work artifacts перенесены в collision-safe системный cache.
- Явные optional artifacts получили stem-prefixed имена, а global public path collisions отклоняются до transcription requests.
- `.gitignore` больше не скрывает `internal/output`; TD-006 закрыт regression coverage.
- `make fmt-check`, `make test`, `make vet`, `make docs-check`, `git diff --check` и race tests прошли.
- `make ci` прошёл с установленным Go 1.24.2 и текущим `golangci-lint`: `0 issues`; live provider tests не запускались, поскольку provider contract не менялся.
