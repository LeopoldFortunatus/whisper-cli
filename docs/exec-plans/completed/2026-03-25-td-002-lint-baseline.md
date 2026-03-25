# TD-002 Lint Baseline

Owner: Platform Team
Last Verified: 2026-03-25
Status: Completed

## Goal

Добавить в `whisper-cli` воспроизводимую lint-проверку как часть локального и CI quality gate.

## Context

- В проекте были `fmt-check`, `vet`, `test` и `docs-check`, но не было отдельного lint-step.
- Исходный `TD-002` фиксировал правильную проблему, но предлагал слишком грубый план: копирование lint-конфига из большого чужого репозитория.
- На машине есть `golangci-lint` v2, а в домашнем каталоге есть устаревший `~/.golangci.yml`, поэтому запуск без локального конфига был нестабилен.

## Risks

1. Прямой перенос `.golangci.yml` из `rbac-interceptor` мог принести лишние project-specific rules и шум.
2. Добавление lint в `make ci` могло сломать pipeline, если baseline не был бы сначала очищен.
3. Непроверяемый запуск через внешний пользовательский конфиг давал бы nondeterministic results между машинами.

## Plan

1. Добавить локальный `golangci-lint` v2 baseline config и явный `make lint`.
2. Включить `lint` в `make ci`, чтобы GitHub Actions автоматически прогонял новый gate.
3. Исправить текущие baseline issues без `nolint`, если они отражают реальный кодовый дефект.
4. Обновить tech-debt tracker и завершить exec-plan.

## Validation

- `golangci-lint run --config .golangci.yml --modules-download-mode=vendor ./...`
- `make ci`

## Decision Log

- 2026-03-25: не копировать целиком lint-конфиг из `rbac-interceptor`; взять только сам подход с `golangci-lint` и настроить минимальный baseline под это репо.
- 2026-03-25: запускать lint с явным `--config`, чтобы не зависеть от пользовательского `~/.golangci.yml`.

## Discoveries

- `golangci-lint run ./...` в этой среде падал из-за домашнего `~/.golangci.yml`, несовместимого с v2.
- Чистый baseline без внешнего конфига показал 3 проблемы: два unchecked `Close` и один unused helper в `groqadapter`.
- Для этого репозитория стандартного baseline набора оказалось достаточно; дополнительная настройка linters пока не нужна.

## Follow-Ups

- При росте проекта можно постепенно ужесточать набор linters вместо одновременного включения большого чужого профиля.

## Retrospective

- `whisper-cli` получил воспроизводимый lint-gate без зависимости от пользовательской среды.
- Качество docs и build-contract синхронизировано: `make lint` задокументирован, `make ci` проверяет lint автоматически, `TD-002` закрыт.
