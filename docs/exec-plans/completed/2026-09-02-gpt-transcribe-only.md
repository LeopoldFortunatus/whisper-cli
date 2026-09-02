# Единственная OpenAI-модель `gpt-transcribe`

Владелец: Platform Team
Проверено: 2026-09-02
Status: Completed

## Цель

Оставить `gpt-transcribe` единственной поддерживаемой и default-моделью OpenAI, полностью удалить diarization-контур и сохранить независимую поддержку моделей других providers.

## Контекст

- OpenAI помечает `gpt-transcribe` как default speech-to-text model.
- До этого среза CLI содержал четыре OpenAI-модели и отдельные ветки для timestamps и diarization.
- Generic `--provider` и `--model` нужны Groq и будущим adapters, поэтому остаются частью CLI-контракта.
- Текущий OpenAI SDK принимает model ID как строку и позволяет передать новый multipart field `languages[]` через extra fields; обновление SDK для этого среза не требуется.

## Риски

1. Старый явный `--model` может незаметно продолжить работать или быть переписан на новый default вместо понятной ошибки.
2. `gpt-transcribe` использует `languages[]`, а не старое singular-поле `language`; неверная сериализация ухудшит языковую подсказку.
3. Удаление diarization может случайно затронуть Groq timestamps, SRT/VTT, multi-input, cache-first output или generic provider abstraction.
4. Unit tests не подтверждают доступность внешнего OpenAI API и валидность реального API key.

## План

1. Свести OpenAI adapter и default config к `gpt-transcribe`, сохранив явную проверку поддерживаемых моделей.
2. Отправлять OpenAI transcription как JSON с `prompt` и одним `languages[]`; не отправлять singular `language`.
3. Удалить diarization из domain, app, provider request, output writer, CLI help, Makefile и документации.
4. Перенести timestamp/SRT/VTT сценарии тестов на Groq-capable fixtures и добавить контрактные тесты новой OpenAI request shape и старых model IDs.
5. Обновить пользовательскую и эксплуатационную документацию.

## Проверка

- targeted `go test` для config, provider, app, cli и output
- `go test -race` для затронутых orchestration/CLI/output пакетов
- `make ci` на Go 1.24.2
- `make build` и проверка `--help`/model completion
- `git diff --check`
- opt-in `make test-live-openai LIVE_AUDIO_FILE=...` остаётся за пределами среза без явного запуска пользователем

## Журнал решений

- 2026-09-02: старые OpenAI model IDs отклоняются явно; автоматической миграции значения `--model` нет.
- 2026-09-02: diarization удаляется полностью, включая public artifact и live target; существующие пользовательские файлы не удаляются.
- 2026-09-02: provider/model abstraction сохраняется для Groq и будущих vendors.
- 2026-09-02: OpenAI SDK не обновляется; `languages[]` сериализуется через поддерживаемый SDK extension point и закрепляется тестом.

## Находки

- Ambient Go в окружении старее версии из `go.mod`; проверки должны использовать локальный Go 1.24.2 и отдельные writable caches.
- Установленный OpenAI SDK уже принимает произвольный model ID и умеет добавлять string multipart fields через `SetExtraFields`, поэтому dependency upgrade не потребовался.
- Чтобы старая OpenAI-модель всегда давала целевую ошибку, capability validation перенесена до credential lookup, provider preflight и проверки `ffmpeg`.

## Следующие шаги

- При необходимости отдельно выполнить opt-in live smoke test с реальным OpenAI API key и аудиофайлом.

## Итоги

- OpenAI default и единственный supported model заменены на `gpt-transcribe`; Groq registry и generic CLI flags сохранены.
- OpenAI adapter отправляет JSON request с `prompt` и одним `languages[]`, не заполняя singular `language`; ответ с `languages` нормализуется в существующее поле `Transcript.Language`.
- Diarization удалён из domain, orchestration, provider request, outputs, CLI help, live tests и документации.
- Старые OpenAI model IDs проверены как explicit errors до runtime preflight и provider request.
- Targeted tests, `go test -race`, `make ci`, `make build`, `--help`, model completion и `git diff --check` прошли на Go 1.24.2.
- Opt-in live OpenAI smoke test не запускался: текущий срез не использует реальный API key и пользовательский audio input.
