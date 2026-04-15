# Архитектура

Владелец: Platform Team
Проверено: 2026-04-14

## Слои

1. `main.go`
- bootstrap CLI
- запуск `cobra` root command
2. `internal/cli`
- дерево команд
- `help/usage`
- `bash completion`
- wiring `flags/env` в `config` и `app`
3. `internal/app`
- оркестрация
- выбор provider'а
- явный `preprocessing step` перед chunking
- планирование чанков
- сборка результата и доставка артефактов
4. `internal/config`
- typed overrides
- резолв `flags > env > defaults`
- runtime validation без CLI parsing
5. `internal/audio`
- интеграция с `ffmpeg/ffprobe`
- поиск media-файлов
- `preprocessing` входов в `m4a`
- генерация чанков
6. `internal/provider/*`
- сборка запросов к конкретному provider'у
- ретраи
- разбор `raw-response`
7. `internal/output`
- сериализация и запись артефактов
8. `internal/domain`
- нормализованные типы transcript'а и модель capabilities
9. `internal/platform/*`
- тонкие обёртки над файловой системой ОС и запуском команд

## Границы

1. `internal/provider/*` не импортируют `internal/app`.
2. `internal/output` не знает о provider SDK.
3. `internal/audio` не знает о transcript-модели и provider'ах.
4. `internal/domain` не зависит от SDK, ОС или парсинга CLI.
5. `internal/config` не тянет `cobra`, filesystem-конфиг, оркестрацию и network logic.
6. `internal/cli` не делает `provider preflight`, не требует `ffmpeg` и не запускает normal transcription flow для `help/completion`.

## Ключевые решения

1. `Provider capability gating` делается до запуска транскрипции.
2. Нормализованные output-артефакты важнее `provider-specific wire shape`.
3. Публичный CLI строится через `cobra`, а `bash completion` генерируется из того же command tree.
4. Runtime contract ограничен `flags > env > defaults`; `YAML`-конфиг в runtime больше не поддерживается.
5. OpenRouter выделен как `adapter slot`, но не включён до появления подтверждённого `transcription contract`.
