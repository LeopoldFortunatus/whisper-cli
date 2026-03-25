# Docs Index

Owner: Platform Team
Last Verified: 2026-03-25

## Knowledge Map

Это точка входа для людей и агентов. Новая договорённость должна появиться в одном source-of-truth документе ниже.

## Core Documents

| Document | Scope |
| --- | --- |
| [`../README.md`](../README.md) | обзор CLI, запуск, env, examples |
| [`ARCHITECTURE.md`](ARCHITECTURE.md) | слои, границы и package ownership |
| [`MAINTENANCE.md`](MAINTENANCE.md) | quality loop, live tests, DoD |
| [`ROADMAP.md`](ROADMAP.md) | продуктовые и инженерные slices |
| [`tech-debt-tracker.md`](tech-debt-tracker.md) | явный реестр долгов и blocked follow-ups |
| [`exec-plans/README.md`](exec-plans/README.md) | навигация по execution plans |

## Maintenance Rules

1. `AGENTS.md` остаётся картой, не энциклопедией.
2. Для длинной задачи useful memory живёт в `docs/exec-plans/`.
3. Документация и качество обновляются вместе с кодом.
4. Минимальный merge gate для локальной работы: `make ci`.
