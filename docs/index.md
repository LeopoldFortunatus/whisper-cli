# Индекс документации

Владелец: Platform Team
Проверено: 2026-03-25

## Карта знаний

Это точка входа для людей и агентов. Новая договорённость должна появиться в одном документе-источнике ниже.

## Основные документы

| Документ | Область |
| --- | --- |
| [`../README.md`](../README.md) | обзор CLI, запуск, `env`, примеры |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | слои, границы и владение пакетами |
| [`MAINTENANCE.md`](MAINTENANCE.md) | quality loop, `live tests`, DoD |
| [`PROJECT_REVIEW_WORKFLOW.md`](PROJECT_REVIEW_WORKFLOW.md) | полное ревью проекта и протокол pre-merge review |
| [`ROADMAP.md`](ROADMAP.md) | продуктовые и инженерные срезы |
| [`tech-debt-tracker.md`](tech-debt-tracker.md) | явный реестр долгов и заблокированных follow-up задач |
| [`exec-plans/README.md`](exec-plans/README.md) | навигация по планам выполнения |

## Правила сопровождения

1. `AGENTS.md` остаётся картой, не энциклопедией.
2. Для длинной задачи полезный контекст живёт в `docs/exec-plans/`.
3. Workflow ревью должен быть стандартизован, а не придуман заново в каждом диалоге.
4. Документация и качество обновляются вместе с кодом.
5. Минимальный локальный `merge gate`: `make ci`.
