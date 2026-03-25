# План выполнения

Владелец: Platform Team
Проверено: 2026-03-25
Status: Completed

## Цель

Провести полное `technical review` проекта, зафиксировать findings по `severity` и подготовить кандидатов в техдолг на основе текущего состояния кода, тестов, `quality loop` и документации.

## Контекст

- `Review` идёт по `Protocol A` из `docs/PROJECT_REVIEW_WORKFLOW.md`.
- `Source-of-truth docs`: `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/tech-debt-tracker.md`.
- Текущий `worktree` грязный в `README.md`; ревью не должно перетирать несвязанные пользовательские изменения.
- `Project scope` включает оркестрацию в `internal/app`, резолв конфигурации, разбиение аудио на чанки, `provider adapters`, нормализованную `domain model`, запись артефактов, тесты и локальные `quality gates`.

## Риски

1. Существующие пользовательские изменения в `README.md` могут создать `docs/reality drift` во время ревью, поэтому findings должны отличать базовые проблемы от `in-flight edits`.
2. Локальный `sandbox` для обычного чтения сломан, поэтому сбор evidence зависит от `escalated local commands`.
3. `make ci` может показать `environment-specific failures`, не связанные с кодом продукта; их нужно отделять от детерминированных проблем репозитория.
4. Поведение `live provider` нельзя считать проверенным, пока явно не запущены `opt-in smoke tests`.

## План

1. Прочитать `review protocol`, архитектуру, документы по сопровождению и техдолгу; зафиксировать `review scope` и проверки.
2. Собрать локальный `quality evidence` через `make ci` и проверить `build/test/docs tooling` на пробелы в воспроизводимости.
3. Просмотреть ключевые `runtime packages` и `provider adapters` на предмет `correctness`, нарушений границ, `contract drift` и нехватки `coverage`.
4. Собрать отчёт ревью с `severity`, `evidence`, `impact`, `proposed fix` и `debt mapping`; при необходимости добавить новых кандидатов в техдолг.

## Проверка

- `make ci`
- `targeted code inspection` для `main.go`, `internal/app`, `internal/config`, `internal/audio`, `internal/provider/*`, `internal/output` и связанных тестов
- проверка согласованности документации с текущими `source-of-truth` документами

## Журнал решений

- 2026-03-25: использовать отдельный `exec-plan`, потому что задача охватывает код, документацию, тесты и классификацию техдолга.

## Находки

- `make ci` зелёный в текущем `workspace`.
- `make build` и `./bin/whisper-cli -h` проходят, поэтому findings ревью не вызваны сломанным `bootstrap path`.
- Текущий локальный `worktree` был грязным только в `README.md`; `review evidence` собирался без изменения этого файла.
- Ревью выявило три конкретных кандидата в техдолг: конфликты `batch output directories`, отсутствие `fail-fast cancellation` для транскрипции чанков и тонкое покрытие `provider contract` вне OpenAI.

## Следующие шаги

- Превратить `TD-006`, `TD-007` и `TD-008` в `implementation slices` с тестами.

## Итоги

- Просмотрены `main.go`, `internal/app`, `internal/config`, `internal/audio`, `internal/output`, `internal/provider/*`, связанные тесты, документация и `quality scripts`.
- Проверены `make ci`, `GOFLAGS=-mod=vendor go test -cover ./...`, `make build` и `./bin/whisper-cli -h`.
- Findings были перенесены в `docs/tech-debt-tracker.md` как новые активные кандидаты в техдолг.
