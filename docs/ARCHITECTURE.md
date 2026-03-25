# Architecture

Owner: Platform Team
Last Verified: 2026-03-25

## Layers

1. `main.go`
- только bootstrap и запуск `internal/app`
2. `internal/app`
- orchestration
- config loading
- provider selection
- chunk scheduling
- merge + delivery
3. `internal/audio`
- ffmpeg/ffprobe integration
- file discovery
- chunk generation
4. `internal/provider/*`
- provider-specific request building
- retries
- raw-response parsing
5. `internal/output`
- serialization and artifact writing
6. `internal/domain`
- normalized transcript types and capability model
7. `internal/platform/*`
- thin wrappers over OS filesystem and command execution

## Boundaries

1. `internal/provider/*` не импортируют `internal/app`.
2. `internal/output` не знает о provider SDK.
3. `internal/audio` не знает о transcript model и providers.
4. `internal/domain` не зависит от SDK, OS или CLI parsing.
5. `internal/config` не тянет orchestration и network logic.

## Design Choices

1. Provider capability gating делается до запуска транскрипции.
2. Normalized outputs важнее provider-specific wire shape.
3. `config.yaml` поддерживается как legacy compatibility, но не как основной runtime contract.
4. OpenRouter выделен как adapter slot, но не включён до появления подтверждённого transcription contract.
