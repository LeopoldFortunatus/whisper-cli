# Архитектура

Владелец: Platform Team
Проверено: 2026-03-27

## Слои

1. `main.go`
- bootstrap CLI
- dispatch для `help/completion`
2. `internal/app`
- оркестрация
- загрузка конфигурации
- выбор provider'а
- явный `preprocessing step` перед chunking
- планирование чанков
- сборка результата и доставка артефактов
3. `internal/config`
- парсинг `flags`
- резолв `flags > env > YAML > defaults`
- единый metadata-source для CLI flag contract
4. `internal/completion`
- генерация `bash completion` script
- использование metadata из `internal/config` и provider model registry без запуска runtime orchestration
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
5. `internal/config` не тянет оркестрацию и network logic.
6. `internal/completion` не делает `provider preflight`, не требует `ffmpeg` и не запускает normal transcription flow.

## Ключевые решения

1. `Provider capability gating` делается до запуска транскрипции.
2. Нормализованные output-артефакты важнее `provider-specific wire shape`.
3. `config.yaml` поддерживается как `legacy compatibility`, но не как основной `runtime contract`.
4. OpenRouter выделен как `adapter slot`, но не включён до появления подтверждённого `transcription contract`.
