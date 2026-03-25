# Agent Harness And Multi-Provider Refactor

Owner: Platform Team
Last Verified: 2026-03-25
Status: Completed

## Goal

Сделать `whisper-cli` агент-легибельным и подготовить кодовую базу к multi-provider speech-to-text развитию.

## Context

- Исходное состояние: монолитный `main.go`, отсутствовали docs map, review workflow, Makefile quality loop, локальный `make ci` и тестовый baseline.
- Целевой результат: thin bootstrap, internal package boundaries, нормализованный transcript model, provider adapters, проверяемый harness слой.

## Risks

1. При разрезании монолита можно было потерять совместимость CLI и legacy YAML-конфига.
2. Provider contracts для OpenAI/Groq могли разойтись с нормализованным output model.
3. Слишком тонкий docs layer дал бы формальный harness без operational clarity.

## Plan

- введены `AGENTS.md`, docs map, `Makefile`, локальный `make ci` и docs-check
- монолитный `main.go` разрезан на internal packages
- добавлены OpenAI/Groq adapters и capability gating
- добавлены unit tests на config, audio discovery, outputs, orchestration и adapter retry path
- закреплён roadmap для subtitle, diarization и OpenRouter follow-up

## Validation

- `go test ./...`
- `make ci`

## Decision Log

- 2026-03-25: использовать `flags > env > YAML > defaults`, сохранив YAML как legacy compatibility layer
- 2026-03-25: держать OpenRouter как blocked adapter slot, а не как half-working implementation
- 2026-03-25: нормализовать outputs на уровне internal domain model, а provider raw хранить отдельно

## Discoveries

- Основной operational gap был не только в коде, но и в repo legibility: roadmap status ambiguity, отсутствие review protocol, слабая docs mechanization.
- OpenAI diarization потребовал отдельный request shape через `diarized_json`.

## Follow-Ups

- OpenRouter остаётся blocked до подтверждённого official transcription contract.
- При росте числа task slices можно ужесточить `docs-check` дальше: stale active plans, richer metadata checks и stricter exec-plan linting.

## Retrospective

- Репозиторий стал usable как system of record, но первая версия harness была слишком тонкой по части roadmap statuses и review protocol.
- Следующий meaningful improvement должен идти в сторону stronger docs automation, а не ещё одного большого AGENTS файла.
