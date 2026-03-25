# Рефакторинг agent harness и multi-provider поддержки

Владелец: Platform Team
Проверено: 2026-03-25
Status: Completed

## Цель

Сделать `whisper-cli` агент-легибельным и подготовить кодовую базу к multi-provider speech-to-text развитию.

## Контекст

- Исходное состояние: монолитный `main.go`, отсутствовали карта документации, workflow ревью, `Makefile quality loop`, локальный `make ci` и тестовый baseline.
- Целевой результат: тонкий `bootstrap`, границы `internal packages`, нормализованная `transcript model`, `provider adapters`, проверяемый `harness` слой.

## Риски

1. При разрезании монолита можно было потерять совместимость CLI и `legacy YAML`-конфига.
2. `Provider contracts` для OpenAI/Groq могли разойтись с нормализованной `output model`.
3. Слишком тонкий слой документации дал бы формальный `harness` без `operational clarity`.

## План

- введены `AGENTS.md`, карта документации, `Makefile`, локальный `make ci` и `docs-check`
- монолитный `main.go` разрезан на `internal packages`
- добавлены `OpenAI/Groq adapters` и `capability gating`
- добавлены `unit tests` на `config`, `audio discovery`, `outputs`, оркестрацию и `adapter retry path`
- закреплена дорожная карта для `subtitle`, `diarization` и `OpenRouter follow-up`

## Проверка

- `go test ./...`
- `make ci`

## Журнал решений

- 2026-03-25: использовать `flags > env > YAML > defaults`, сохранив YAML как `legacy compatibility layer`
- 2026-03-25: держать OpenRouter как `blocked adapter slot`, а не как `half-working implementation`
- 2026-03-25: нормализовать `outputs` на уровне `internal domain model`, а `provider raw` хранить отдельно

## Находки

- Основной `operational gap` был не только в коде, но и в читаемости репозитория: неоднозначные `roadmap statuses`, отсутствие `review protocol`, слабая механизация документации.
- OpenAI `diarization` потребовал отдельный `request shape` через `diarized_json`.

## Следующие шаги

- OpenRouter остаётся заблокированным до подтверждённого `official transcription contract`.
- При росте числа `task slices` можно ужесточить `docs-check` дальше: `stale active plans`, `richer metadata checks` и более строгий `exec-plan linting`.

## Итоги

- Репозиторий стал пригоден как `system of record`, но первая версия `harness` была слишком тонкой по части `roadmap statuses` и `review protocol`.
- Следующий существенный шаг должен идти в сторону более сильной автоматизации документации, а не ещё одного большого файла `AGENTS`.
