# AGENTS Map

Owner: Platform Team
Last Verified: 2026-03-25

Этот файл - короткая карта. Истина живёт в `docs/` и рядом с кодом.

## Workflow

1. Read:
- `README.md`
- `docs/index.md`
- релевантный файл в `docs/exec-plans/`
 - `docs/PROJECT_REVIEW_WORKFLOW.md` если задача про review
2. Plan:
- для длинной задачи создай или обнови exec-plan
- зафиксируй риски и checks до кодовых правок
3. Implement:
- меняй код малыми тематическими шагами
- не смешивай refactor, provider changes и docs drift без причины
4. Validate:
- `make ci`
- внешнего CI нет; quality gate сейчас только локальный `make ci`
- для provider-рисков при необходимости запускай opt-in live smoke tests
5. Deliver:
- короткий changelog
- риски и что проверено
- completed exec-plan переносится в `docs/exec-plans/completed/`

## Source Of Truth

- карта знаний: `docs/index.md`
- архитектура: `docs/ARCHITECTURE.md`
- эксплуатация и DoD: `docs/MAINTENANCE.md`
- review workflow: `docs/PROJECT_REVIEW_WORKFLOW.md`
- roadmap: `docs/ROADMAP.md`
- техдолг: `docs/tech-debt-tracker.md`
- планы исполнения: `docs/exec-plans/`
- обзор и запуск: `README.md`

## Golden Rules

1. `main.go` только bootstrap.
2. Оркестрация живёт в `internal/app`.
3. Provider adapters не тянут orchestration внутрь себя.
4. Normalized transcript model живёт в `internal/domain`.
5. Артефакты пишет только `internal/output`.
6. Конфиг резолвится через `flags > env > YAML > defaults`.
7. Runtime output и локальные config/env файлы не хранятся в git-tracked state.
8. Новое правило должно быть проверяемым тестом или `make`-gate.
