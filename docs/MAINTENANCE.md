# Эксплуатация

Владелец: Platform Team
Проверено: 2026-09-02

## Локальный build contract

- `make build` пишет бинарник в `./bin/whisper-cli`
- `make install` копирует `./bin/whisper-cli` в `~/.local/bin/whisper-cli` и создаёт `~/.local/bin`, если каталога ещё нет
- для непостоянной установки или проверки можно переопределить `INSTALL_DIR`, например `make install INSTALL_DIR=/tmp/whisper-cli-bin`
- `make install-bash-completion` пишет completion script в `~/.local/share/bash-completion/completions/whisper-cli` и создаёт каталог при необходимости
- для проверки completion install без записи в домашний каталог можно переопределить `BASH_COMPLETION_DIR`, например `make install-bash-completion BASH_COMPLETION_DIR=/tmp/whisper-cli-completion`
- разовый вариант без install target: `source <(./bin/whisper-cli completion bash)`
- runtime config резолвится только через `flags > env > defaults`; `config.yaml` больше не участвует в CLI contract
- default public output — `<source-stem>.txt` рядом с source; `--output-dir`/`WHISPER_CLI_OUTPUT_DIR` являются явным override
- multi-file запуск использует `whisper-cli --input *.m4a`; shell expansion order сохраняется между files
- служебные artifacts хранятся в `os.UserCacheDir()/whisper-cli/<input-path-hash>/`, а повторный запуск очищает только cache конкретного input
- cache является workspace последнего запуска, а не memoization: каждый запуск повторяет provider requests
- OpenAI API key в обычном Linux runtime резолвится через Secret Service: `secret-tool lookup application whisper-cli provider openai kind api-key`
- `OPENAI_API_KEY` остаётся fallback для CI и opt-in live smoke tests, если Secret Service недоступен или запись не найдена
- `whisper-cli --help` и `whisper-cli completion bash` не требуют `API key`, `secret-tool`, `ffmpeg` или provider preflight
- единственная поддерживаемая OpenAI-модель и OpenAI default — `gpt-transcribe`; старые явные model IDs завершают запуск ошибкой

## Локальный quality loop

Для этого проекта нет внешнего CI. Официальный `quality gate` сейчас только локальный `make ci`.

- `make fmt`
- `make fmt-check`
- `make lint`
- `make test`
- `make vet`
- `make docs-check`
- `make ci`

## Live smoke tests

Запускаются только по `opt-in`. В `make ci` не входят.

- `make test-live-openai LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-groq LIVE_AUDIO_FILE=/abs/path/audio.m4a`

Требования:

- для OpenAI: Secret Service или fallback `OPENAI_API_KEY`
- для Groq: `GROQ_API_KEY`
- `LIVE_AUDIO_FILE` должен указывать на локальный аудиофайл

Для env fallback ключи должны быть экспортированы в окружение дочернего процесса. `echo $OPENAI_API_KEY` может печатать shell-переменную, которая не видна `whisper-cli`, если не выполнен `export`.

## Критерии готовности

1. Код и документация согласованы.
2. `make ci` green.
3. Для новых provider-фич есть `unit coverage`.
4. Если менялся `provider contract`, `live smoke tests` либо запущены, либо явно отмечены как `not run` с причиной.
5. Если задача была длинной или многосрезовой, есть self-contained `exec-plan`.
