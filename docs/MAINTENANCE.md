# Maintenance

Owner: Platform Team
Last Verified: 2026-03-25

## Local Build Contract

- `make build` пишет бинарник в `./bin/whisper-cli`
- `make install` копирует `./bin/whisper-cli` в `~/.local/bin/whisper-cli` и создаёт `~/.local/bin`, если каталога ещё нет
- для непостоянной установки или проверки можно переопределить `INSTALL_DIR`, например `make install INSTALL_DIR=/tmp/whisper-cli-bin`

## Local Quality Loop

Для этого проекта нет внешнего CI. Официальный quality gate сейчас только локальный `make ci`.

- `make fmt`
- `make fmt-check`
- `make lint`
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

Ключи должны быть экспортированы в environment дочернего процесса. `echo $OPENAI_API_KEY` может печатать shell-переменную, которая не видна `whisper-cli`, если не выполнен `export`.

## Definition Of Done

1. Код и docs согласованы.
2. `make ci` green.
3. Для новых provider-фич есть unit coverage.
4. Если менялся provider contract, live smoke tests либо запущены, либо явно not run с причиной.
5. Если задача была длинной или многосрезовой, есть self-contained exec-plan.
