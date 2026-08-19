# Полное ревью проекта и техдолг

Владелец: Platform Team
Проверено: 2026-06-05
Status: Completed

## Цель

Провести полное `technical review` проекта по `docs/PROJECT_REVIEW_WORKFLOW.md`, проверить текущий локальный `quality gate`, сверить код с архитектурными правилами и перенести подтверждённые follow-up проблемы в `docs/tech-debt-tracker.md`.

## Контекст

- Scope: `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/ROADMAP.md`, `docs/tech-debt-tracker.md`, `Makefile`, `main.go`, `internal/cli`, `internal/app`, `internal/config`, `internal/audio`, `internal/output`, `internal/provider/*`, `internal/credentials` и tests.
- Review опирался на Go checklist для CLI, style и testing: `cobra` command behavior, stdout/stderr discipline, `flags > env > defaults`, package boundaries, provider contract coverage, binary smoke и cancellation/error paths.
- Worktree перед ревью был чистым; ignored runtime artifacts (`.env`, `config.yaml`, `transcribe/`, `bin/`) не находятся в `git-tracked state`.

## Риски

1. `make ci` сейчас не является зелёным из-за локального toolchain/linter mismatch, поэтому lint-сигнал не подтверждён.
2. `live smoke tests` не запускались: OpenAI/Groq remote behavior и реальные ключи вне scope без явного opt-in.
3. Часть выявленных проблем уже есть в `Active` техдолге; review не должен создавать дублирующие TD.

## План

1. Прочитать source-of-truth docs и предыдущие completed review plans.
2. Запустить официальный `make ci`, затем разложить его на отдельные checks с repo-compatible Go toolchain.
3. Проверить binary-level CLI smoke для `--help` и `completion bash`.
4. Просмотреть архитектурные границы и существующие active debt items на актуальность.
5. Добавить только новые debt items и зафиксировать review evidence в completed exec-plan.

## Проверка

- `go env GOROOT GOVERSION GOTOOLCHAIN` - ambient environment: `GOROOT=/home/arykalin/go/go1.23.3`, `GOVERSION=go1.23.3`, `GOTOOLCHAIN=local`.
- `/home/arykalin/bin/golangci-lint version` - `golangci-lint 2.3.0 built with go1.24.4`.
- `make ci` - failed in `make lint`: `go.mod requires go >= 1.24.2 (running go 1.23.3; GOTOOLCHAIN=local)`.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make ci` - failed in `golangci-lint`: `panic: file requires newer Go version go1.26 (application built with go1.24)`.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make fmt-check` - passed.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make vet` - passed.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make test` - passed.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make docs-check` - passed.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GOFLAGS=-mod=vendor /home/arykalin/go/go1.26.3/bin/go test -race ./...` - passed.
- `env GOROOT=/home/arykalin/go/go1.26.3 PATH=/home/arykalin/go/go1.26.3/bin:/home/arykalin/bin:/usr/local/bin:/usr/bin:/bin GO=/home/arykalin/go/go1.26.3/bin/go make build` - passed.
- `./bin/whisper-cli --help` - passed; no API key, `secret-tool`, `ffmpeg` or provider preflight required.
- `./bin/whisper-cli completion bash` - passed; generated bash completion to `stdout`.
- `git ls-files .env config.yaml 'transcribe/**' 'bin/**'` - no tracked runtime artifacts found.

## Журнал решений

- 2026-06-05: `make ci` failure классифицирован как новый toolchain reproducibility debt, а не как product code failure: tests/race/vet/docs/build проходят с корректным Go 1.26.3.
- 2026-06-05: не заводить новые TD для уже подтверждённых проблем `TD-005`, `TD-006`, `TD-007`, `TD-008`, `TD-011`; они остаются актуальными debt entries.
- 2026-06-05: `live smoke tests` не запускать без explicit opt-in, потому что они требуют provider credentials и локальный `LIVE_AUDIO_FILE`.

## Находки

### High: локальный `make ci` не воспроизводится из-за toolchain/linter mismatch

- Evidence: `Makefile` запускает `$(GOLANGCI_LINT) run ...`; ambient `go` равен `go1.23.3` при `go.mod` `go 1.24.2`, а установленный `/home/arykalin/bin/golangci-lint 2.3.0` built with `go1.24.4` паникует при запуске под `GOROOT=/home/arykalin/go/go1.26.3`.
- Impact: единственный официальный локальный `quality gate` красный по причине окружения; lint-регрессии нельзя отделить от baseline failure.
- Proposed fix: закрепить repo-local compatible toolchain path или version guard для `go`/`golangci-lint`, обновить локальный linter и сделать ошибку preflight понятной.
- Debt: `TD-012`.

### Medium: env integer parsing hides invalid config

- Evidence: `internal/config/config.go` использует `chooseInt`, который возвращает fallback, если env value присутствует, но `strconv.Atoi` завершился ошибкой.
- Impact: misconfigured `WHISPER_CLI_CHUNK_SECONDS` или `WHISPER_CLI_CONCURRENCY` может silently изменить runtime duration и provider cost.
- Proposed fix: сделать error-returning path для invalid numeric env overrides.
- Debt: `TD-011`.

### Medium: batch output collisions remain unresolved

- Evidence: `internal/app/app.go` строит `fileOutputDir` из basename без extension; directory input с `lecture.m4a` и `lecture.mp3` попадёт в один `<output-dir>/lecture`.
- Impact: возможны overwrite и смешивание `_work` artifacts.
- Proposed fix: сделать collision-safe naming для directory input и добавить regression test.
- Debt: `TD-006`.

### Medium: chunk transcription still does not fail fast

- Evidence: `internal/app/app.go` enqueue-ит все chunks, ждёт `wg.Wait()`, затем читает `results` и только после этого возвращает первую ошибку.
- Impact: после первой ошибки queued/in-flight provider requests продолжают выполняться.
- Proposed fix: использовать `context.WithCancel`, прекращать scheduling после первой ошибки и покрыть `cancel/error path` тестами.
- Debt: `TD-007`.

### Low: binary-level smoke coverage still incomplete

- Evidence: ручные `./bin/whisper-cli --help` и `./bin/whisper-cli completion bash` проходят, но debt item про стабильный binary-level smoke остаётся активным.
- Impact: часть entrypoint/exit-code regressions может быть поймана только ручной проверкой.
- Proposed fix: добавить lightweight binary smoke tests без live provider dependency.
- Debt: `TD-005`.

### Low: provider contract coverage remains thin outside OpenAI

- Evidence: `internal/provider` helper layer и часть Groq request/response behavior остаются без deterministic unit coverage.
- Impact: drift в retry/parser/raw marshaling/Groq request construction может пройти через локальные tests.
- Proposed fix: добавить tests для `Retry`, `ParseOpenAICompatibleTranscript`, `MarshalRawArray` и Groq request/response construction.
- Debt: `TD-008`.

## Следующие шаги

- Первым брать `TD-012`, потому что без зелёного `make ci` проект не имеет единого локального merge gate.
- После восстановления gate брать `TD-011`, `TD-006`, `TD-007`, `TD-005`, `TD-008` отдельными implementation slices.
- `RM-012` по Groq credentials оставлять отдельным roadmap item и не смешивать с текущим review cleanup.

## Итоги

- Review завершён без production code changes.
- Добавлен новый debt item `TD-012`.
- Подтверждены как актуальные existing debt items: `TD-011`, `TD-006`, `TD-007`, `TD-005`, `TD-008`.
- `make ci` остаётся red из-за toolchain/linter mismatch; остальные локальные checks проходят при явном `GOROOT=/home/arykalin/go/go1.26.3`.
