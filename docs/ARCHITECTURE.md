# Архитектура

Владелец: Platform Team
Проверено: 2026-03-25

## Слои

1. `main.go`
- только bootstrap и запуск `internal/app`
2. `internal/app`
- оркестрация
- загрузка конфигурации
- выбор provider'а
- планирование чанков
- сборка результата и доставка артефактов
3. `internal/audio`
- интеграция с `ffmpeg/ffprobe`
- поиск файлов
- генерация чанков
4. `internal/provider/*`
- сборка запросов к конкретному provider'у
- ретраи
- разбор `raw-response`
5. `internal/output`
- сериализация и запись артефактов
6. `internal/domain`
- нормализованные типы transcript'а и модель capabilities
7. `internal/platform/*`
- тонкие обёртки над файловой системой ОС и запуском команд

## Границы

1. `internal/provider/*` не импортируют `internal/app`.
2. `internal/output` не знает о provider SDK.
3. `internal/audio` не знает о transcript-модели и provider'ах.
4. `internal/domain` не зависит от SDK, ОС или парсинга CLI.
5. `internal/config` не тянет оркестрацию и network logic.

## Ключевые решения

1. `Provider capability gating` делается до запуска транскрипции.
2. Нормализованные output-артефакты важнее `provider-specific wire shape`.
3. `config.yaml` поддерживается как `legacy compatibility`, но не как основной `runtime contract`.
4. OpenRouter выделен как `adapter slot`, но не включён до появления подтверждённого `transcription contract`.
