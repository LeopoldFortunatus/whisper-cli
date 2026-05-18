# Полное ревью проекта

Владелец: Platform Team
Проверено: 2026-05-18
Status: Completed

## Цель

Провести полное `code review` проекта по архитектуре, документации и качеству Go-кода, зафиксировать findings по `severity` и перенести follow-up задачи в durable docs.

## Контекст

- Review выполнялся по `Protocol A` из `docs/PROJECT_REVIEW_WORKFLOW.md`.
- Scope: `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/ROADMAP.md`, `docs/tech-debt-tracker.md`, `main.go`, `internal/app`, `internal/cli`, `internal/config`, `internal/audio`, `internal/output`, `internal/provider/*`, tests и локальный `quality gate`.
- Go review опирался на практики `golang-cli`, `golang-testing`, `golang-code-style`, `golang-project-layout` и `golang-documentation`.

## Риски

1. `make ci` сейчас красный, поэтому любые новые изменения нельзя считать merge-ready до восстановления локального gate.
2. `live smoke tests` не запускались: review не подтверждает актуальный remote provider behavior.
3. Provider API contracts могут дрейфовать без локального сигнала, если deterministic tests покрывают только часть request/parse/retry behavior.

## План

1. Сверить source-of-truth docs с реальной структурой и runtime contract.
2. Запустить `make ci` и зафиксировать quality evidence.
3. Просмотреть архитектурные границы CLI/app/config/audio/output/provider packages.
4. Проверить test coverage на критичных CLI/provider/orchestration сценариях.
5. Обновить `docs/tech-debt-tracker.md` для новых findings и связать review report с существующими debt items.

## Проверка

- `make ci` - failed: `internal/config.TestResolveUsesDefaultsWithoutEnv` ожидает `whisper-1`, код возвращает актуальный default `gpt-4o-transcribe`.
- `targeted code inspection` для `main.go`, `internal/app`, `internal/cli`, `internal/config`, `internal/audio`, `internal/output`, `internal/provider/*`.
- `docs/source-of-truth inspection` для `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/ROADMAP.md`, `docs/tech-debt-tracker.md`.

## Журнал решений

- 2026-05-18: default OpenAI model считается `gpt-4o-transcribe`; failing test классифицирован как stale test/gate failure, а не как production default bug.
- 2026-05-18: review не чинит code path; новые проблемы фиксируются как `TD-010` и `TD-011`, старые подтверждённые проблемы остаются в `TD-006`, `TD-007`, `TD-008`.

## Находки

### High: `make ci` не является зелёным quality gate

- Evidence: `make ci` падает в `internal/config.TestResolveUsesDefaultsWithoutEnv`; test ожидает `whisper-1`, а `defaultModelForProvider` возвращает `gpt-4o-transcribe`.
- Impact: официальный локальный merge gate из `README.md`/`MAINTENANCE.md` сейчас не воспроизводится; future review не может отделить новые регрессии от baseline failure.
- Proposed fix: обновить stale assertion, явно задокументировать default model и добавить проверку на синхронность code/test/docs при смене defaults.
- Debt: `TD-010`.

### Medium: env integer parsing hides invalid config

- Evidence: `chooseInt` возвращает fallback, если env value присутствует, но `strconv.Atoi` завершился ошибкой.
- Impact: пользователь может запустить транскрипцию с неожиданным `chunk-seconds` или `concurrency`; это влияет на runtime duration, provider cost и troubleshooting.
- Proposed fix: возвращать configuration error для invalid numeric env overrides, сохранив defaults только для отсутствующих env variables.
- Debt: `TD-011`.

### Medium: batch output collisions still unresolved

- Evidence: `processFile` строит `fileOutputDir` из `strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))`.
- Impact: directory input с `lecture.m4a` и `lecture.mp3` пишет в один `<output-dir>/lecture`, создавая риск overwrite/mixed `_work` artifacts.
- Proposed fix: сделать collision-safe naming для directory input и добавить directory-level regression test.
- Debt: `TD-006`.

### Medium: chunk transcription still waits for all queued work after first error

- Evidence: `transcribeChunks` ставит все chunks в buffered `jobs`, ждёт `wg.Wait()` и только потом читает `results`/возвращает первый error.
- Impact: после первой ошибки уже queued/in-flight requests продолжают выполняться; пользователь платит временем и potentially provider calls.
- Proposed fix: перейти на `context.WithCancel`, прекращать scheduling после первой ошибки и читать results concurrently.
- Debt: `TD-007`.

### Low: provider-helper and Groq deterministic coverage remain thin

- Evidence: `internal/provider` helper functions не имеют package tests; `groqadapter` покрывает только preflight и sorted models, тогда как OpenAI adapter имеет retry/request-shape tests.
- Impact: drift в parser/retry/raw marshaling/Groq request construction может пройти локальные тесты.
- Proposed fix: добавить tests для `Retry`, `ParseOpenAICompatibleTranscript`, `MarshalRawArray` и Groq request/response parsing.
- Debt: `TD-008`.

## Следующие шаги

- Первым implementation slice восстановить `make ci` через `TD-010`.
- Затем отдельно брать `TD-011`, `TD-006`, `TD-007`, `TD-008`; не смешивать behavior fixes и provider changes.
- Live smoke tests запускать только opt-in с ключами и `LIVE_AUDIO_FILE`.

## Итоги

- Review завершён без production code changes.
- Добавлены новые debt items: `TD-010`, `TD-011`.
- Existing debt items `TD-006`, `TD-007`, `TD-008` подтверждены как актуальные.
- Local quality gate status: red until `TD-010` is fixed.
