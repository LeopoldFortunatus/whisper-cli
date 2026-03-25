# Exec Plan

Owner: Platform Team
Last Verified: 2026-03-25
Status: Completed

## Goal

Провести full-project technical review, зафиксировать findings по severity и подготовить кандидатов в техдолг на основе текущего состояния кода, тестов, quality loop и документации.

## Context

- Review follows `docs/PROJECT_REVIEW_WORKFLOW.md` Protocol A.
- Source-of-truth docs: `README.md`, `docs/index.md`, `docs/ARCHITECTURE.md`, `docs/MAINTENANCE.md`, `docs/tech-debt-tracker.md`.
- Current worktree is dirty in `README.md`; review must not overwrite unrelated user changes.
- Project scope includes orchestration in `internal/app`, config resolution, audio chunking, provider adapters, normalized domain model, artifact writing, tests, and local quality gates.

## Risks

1. Existing user changes in `README.md` can create docs/reality drift during review, so findings must distinguish baseline issues from in-flight edits.
2. Local environment sandbox is broken for ordinary reads, so evidence collection depends on escalated local commands.
3. `make ci` may surface environment-specific failures unrelated to product code; those need to be separated from deterministic repo issues.
4. Live provider behavior cannot be treated as verified unless opt-in smoke tests are explicitly run.

## Plan

1. Read review protocol, architecture, maintenance, and debt docs; capture review scope and checks.
2. Run local quality evidence (`make ci`) and inspect build/test/docs tooling for reproducibility gaps.
3. Review core runtime packages and provider adapters for correctness, boundary violations, contract drift, and missing coverage.
4. Produce a review report with severity, evidence, impact, proposed fix, and debt mapping; add new debt candidates if warranted.

## Validation

- `make ci`
- targeted code inspection of `main.go`, `internal/app`, `internal/config`, `internal/audio`, `internal/provider/*`, `internal/output`, and related tests
- docs consistency checks against current source-of-truth documents

## Decision Log

- 2026-03-25: Using a dedicated exec-plan because the task spans code, docs, tests, and tech-debt classification.

## Discoveries

- `make ci` is green in the current workspace.
- `make build` and `./bin/whisper-cli -h` succeed, so the review findings are not caused by a broken bootstrap path.
- Current local worktree was dirty only in `README.md`; review evidence was collected without modifying that file.
- Review identified three concrete debt candidates: batch output directory collisions, lack of fail-fast cancellation for chunk transcription, and thin provider-contract coverage outside OpenAI.

## Follow-Ups

- Convert TD-006, TD-007, and TD-008 into implementation slices with tests.

## Retrospective

- Reviewed `main.go`, `internal/app`, `internal/config`, `internal/audio`, `internal/output`, `internal/provider/*`, related tests, docs, and quality scripts.
- Verified `make ci`, `GOFLAGS=-mod=vendor go test -cover ./...`, `make build`, and `./bin/whisper-cli -h`.
- Findings were mapped into `docs/tech-debt-tracker.md` as new active debt candidates.
