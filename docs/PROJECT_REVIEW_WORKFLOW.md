# Регламент ревью проекта

Владелец: Platform Team
Проверено: 2026-03-25

## Назначение

Этот документ стандартизует два сценария review:

1. полное ревью проекта
2. `pre-merge review` текущей задачи

## Протокол A: полное ревью проекта

### Предусловия

1. Прочитать `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`.
2. Если review крупный, создать или обновить `exec-plan`.
3. Зафиксировать checks, risks и scope ревью до findings.

### Обязательные проверки

1. `README` против реального поведения.
2. Согласованность `AGENTS/docs/source-of-truth`.
3. Воспроизводимость локального `quality loop` через `make ci`.
4. Нарушения архитектурных границ.
5. `Dead code` и устаревшая документация.
6. Drift в `provider contract` и отсутствующие тесты.

### Формат отчёта

Для каждого finding:

- `severity`: `critical` | `high` | `medium` | `low`
- подтверждение через `file/path`
- влияние
- предлагаемый фикс
- привязка к техдолгу, если применимо

## Протокол B: pre-merge review

### Предусловия

1. Определить `task slice` или `exec-plan`.
2. Проверить релевантные документы и `acceptance intent`.
3. Собрать `quality evidence`.

### Обязательные проверки

1. `make ci`
2. `targeted tests` по изменённой зоне
3. согласованность `docs/config`
4. если менялся `provider contract`: `live smoke tests` с отметкой `run/not run` и причиной

### Решение

Выход review должен заканчиваться явным `verdict`:

- `GO`
- `NO-GO`

## Политика severity

1. `critical` / `high` по `correctness`, `contract drift`, `data loss` или `security risk` => `NO-GO`
2. `medium` => допускается только при явном `follow-up` плане
3. `low` => не блокирует merge
