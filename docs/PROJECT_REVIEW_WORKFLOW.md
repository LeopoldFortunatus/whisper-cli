# Project Review Workflow

Owner: Platform Team
Last Verified: 2026-03-25

## Purpose

Этот документ стандартизует два сценария review:

1. full project review
2. pre-merge review текущей задачи

## Protocol A: Full Project Review

### Preconditions

1. Прочитать `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`.
2. Если review крупный, создать или обновить exec-plan.
3. Зафиксировать checks, risks и review scope до findings.

### Required Checks

1. README vs reality.
2. AGENTS/docs/source-of-truth consistency.
3. Quality loop and CI reproducibility.
4. Architecture boundary violations.
5. Dead code and stale docs.
6. Provider contract drift and missing tests.

### Report Shape

Для каждого finding:

- severity: `critical` | `high` | `medium` | `low`
- file/path evidence
- impact
- proposed fix
- debt mapping if applicable

## Protocol B: Pre-Merge Review

### Preconditions

1. Определить task slice или exec-plan.
2. Проверить relevant docs и acceptance intent.
3. Собрать quality evidence.

### Required Checks

1. `make ci`
2. targeted tests по изменённой зоне
3. docs/config consistency
4. если менялся provider contract: live smoke tests run/not run с причиной

### Decision

Выход review должен заканчиваться явным verdict:

- `GO`
- `NO-GO`

## Severity Policy

1. `critical` / `high` correctness, contract drift, data loss или security risk => `NO-GO`
2. `medium` => допускается только при явном follow-up плане
3. `low` => не блокирует merge
