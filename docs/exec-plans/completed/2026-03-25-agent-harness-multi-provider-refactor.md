# Agent Harness And Multi-Provider Refactor

Owner: Platform Team
Last Verified: 2026-03-25
Status: Completed

## Goal

Сделать `whisper-cli` агент-легибельным и подготовить кодовую базу к multi-provider speech-to-text развитию.

## Changes

- введены `AGENTS.md`, docs map, `Makefile`, CI и docs-check
- монолитный `main.go` разрезан на internal packages
- добавлены OpenAI/Groq adapters и capability gating
- добавлены unit tests на config, audio discovery, outputs, orchestration и adapter retry path
- закреплён roadmap для subtitle, diarization и OpenRouter follow-up

## Validation

- `go test ./...`
- `make ci`
