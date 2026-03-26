# План выполнения

Владелец: Platform Team
Проверено: 2026-03-25
Status: Completed

## Цель

Добавить явный `preprocessing`-шаг перед chunking: поддерживаемые non-`m4a` аудио- и видеофайлы автоматически конвертируются в `m4a` через `ffmpeg`, после чего проходят текущий pipeline транскрипции.

## Контекст

- Затронуты `internal/audio`, `internal/app`, `internal/config`, `README.md`, `docs/ARCHITECTURE.md`, `docs/ROADMAP.md` и `unit tests`.
- До RM-009 CLI уже принимал контейнеры вроде `.mp4` и `.webm`, но не фиксировал отдельный `preprocessing`-контракт и сразу резал вход через `ffmpeg`.
- Промежуточные файлы должны были стать явными и жить в `<output>/<base>/_work/`, а итоговые артефакты должны были остаться в корне `<output>/<base>/`.

## Риски

1. Случайно изменить `provider contract` или `transcript.Source`, хотя preprocessing должен был остаться внутренней деталью `audio/app` слоя.
2. Смешать промежуточные файлы с итоговыми артефактами и сделать layout менее предсказуемым для пользователя.
3. Ослабить покрытие вокруг `ffmpeg`-ошибок и потерять stderr в местах, где он нужен для диагностики.

## План

1. Добавить в `internal/audio` явный `PrepareInput` и переименовать scan-контракт в media-термины.
2. Перевести `internal/app` на явный `_work`-layout и последовательность `PrepareInput -> PrepareChunks -> provider`.
3. Обновить `unit tests` на preprocessing, chunking и orchestration path.
4. Синхронизировать README и docs с новым `media preprocessing`-контрактом.
5. Прогнать `make ci`, после чего перенести plan в `completed/` и отметить RM-009 как `Done`.

## Проверка

- `GOFLAGS=-mod=vendor go test ./internal/audio`
- `GOFLAGS=-mod=vendor go test ./internal/app`
- `make ci`
- `live provider tests`: not run, потому что RM-009 меняет `internal/app` preprocessing path, а не provider adapters

## Журнал решений

- 2026-03-25: scope v1 ограничен текущим набором уже поддерживаемых расширений; новые media extensions остаются вне этой задачи.
- 2026-03-25: промежуточные артефакты сохраняются в `<output>/<base>/_work/`, а не удаляются после выполнения.
- 2026-03-25: `transcribe/test/test.ogg` скопирован в `testdata/media/test.ogg` как локальный smoke asset в нормальном testdata-path.

## Находки

- `internal/audio` уже содержал список поддерживаемых расширений, включая `.mp4` и `.webm`, но naming всё ещё говорил только про `audio`.
- `provider live tests` покрывают adapters напрямую и не проверяют новый preprocessing path в `internal/app`.
- `docs/tech-debt-tracker.md` фиксирует отдельный риск basename-collision для directory mode; эта задача его не закрывает.

## Следующие шаги

- Отдельно решить `TD-006` про collision-safe naming для batch input.
- При необходимости добавить бинарный `e2e smoke`, который проверяет preprocessing через реальный CLI path с использованием `testdata/media/test.ogg`.

## Итоги

- В `internal/audio` добавлен явный `PrepareInput`, который сохраняет non-`m4a` входы как `<output>/<base>/_work/source.m4a`, а chunking теперь всегда работает в `_work/`.
- `internal/app` теперь делает явную последовательность `PrepareInput -> PrepareChunks`, при этом `transcript.Source` сохраняет оригинальный путь пользователя.
- README, architecture и roadmap синхронизированы с новым `media preprocessing`-контрактом, а `make ci` прошёл успешно.
