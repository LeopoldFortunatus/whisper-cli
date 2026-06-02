# RM-010 Secure OpenAI API Key Resolution

Владелец: Platform Team
Проверено: 2026-06-02
Status: Completed

## Цель

Сделать безопасный Linux runtime contract для OpenAI API key: `whisper-cli` читает ключ через Secret Service (`secret-tool`) и использует `OPENAI_API_KEY` только как fallback для CI и opt-in live smoke tests.

## Контекст

- Сейчас `OPENAI_API_KEY` читается в `app.NewDefault()`, из-за чего credential lookup связан с bootstrap/metadata client creation.
- `--help` и `completion bash` не должны запускать `secret-tool`, `ffmpeg` или provider preflight.
- Текущий срез покрывает только OpenAI; Groq остаётся на env-only contract.

## Риски

1. Запуск `secret-tool` в help/completion path сломает lightweight CLI UX.
2. Ошибка отсутствующего ключа может случайно вывести secret value.
3. Provider metadata для completion может начать зависеть от runtime credentials.

## План

1. Добавить read-only resolver для OpenAI key: Secret Service через `secret-tool lookup`, затем `OPENAI_API_KEY`.
2. Перенести OpenAI key resolution из `NewDefault()` в normal `Application.Run` path после выбора provider и до provider preflight.
3. Добавить unit tests для resolver и CLI/app behavior.
4. Обновить `README.md`, `docs/MAINTENANCE.md`, `docs/ROADMAP.md`.
5. Запустить `make ci`; live smoke tests не запускать по умолчанию.

## Проверка

- `make ci`
- live smoke tests: `not run`, потому что меняется local secret resolution, а не provider wire contract.

## Журнал решений

- 2026-06-02: выбран Secret Service через внешний `secret-tool`, чтобы сохранить простой Linux-specific contract без новой Go dependency.
- 2026-06-02: priority order для OpenAI key: Secret Service first, затем `OPENAI_API_KEY`.
- 2026-06-02: `OPENAI_API_KEY` остаётся fallback для CI/live smoke tests.

## Находки

- `golangci-lint` использует `go` из `PATH`, поэтому для локального gate нужен toolchain, совместимый с `go.mod`; в текущем окружении рабочий вариант: Go 1.24.2 в `PATH`.
- Из-за restricted filesystem для успешного lint нужны writable caches (`GOCACHE`, `GOMODCACHE`, `GOLANGCI_LINT_CACHE`).

## Следующие шаги

- Отдельный planned follow-up: secure key resolution для Groq.

## Итоги

- Добавлен read-only OpenAI key resolver: Secret Service через `secret-tool lookup application whisper-cli provider openai kind api-key`, затем fallback к `OPENAI_API_KEY`.
- `app.NewDefault()` больше не читает `OPENAI_API_KEY`; OpenAI runtime provider создаётся после resolution в normal `Application.Run` path.
- `--help` и `completion bash` покрыты тестами на отсутствие OpenAI key resolution.
- README, maintenance docs и roadmap синхронизированы с новым OpenAI contract и Groq follow-up.
- Проверено: `make ci`.
- Live smoke tests: `not run`, потому что менялся local secret resolution, а не provider wire contract.
