# Maintenance

Owner: Platform Team
Last Verified: 2026-03-25

## Local Quality Loop

- `make fmt`
- `make fmt-check`
- `make test`
- `make vet`
- `make docs-check`
- `make ci`

## Live Smoke Tests

Opt-in only. В `make ci` не входят.

- `make test-live-openai LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-openai-diarize LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-groq LIVE_AUDIO_FILE=/abs/path/audio.m4a`

Требования:

- для OpenAI: `OPENAI_API_KEY`
- для Groq: `GROQ_API_KEY`
- `LIVE_AUDIO_FILE` должен указывать на локальный аудиофайл

## Definition Of Done

1. Код и docs согласованы.
2. `make ci` green.
3. Для новых provider-фич есть unit coverage.
4. Если менялся provider contract, live smoke tests либо запущены, либо явно not run с причиной.
