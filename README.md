# whisper-cli

Go CLI для локальной транскрипции медиафайлов и директорий через OpenAI-compatible speech-to-text providers.

## Что делает CLI

- нормализует поддерживаемые аудио- и видеоформаты в `m4a` через `ffmpeg` перед транскрипцией
- режет подготовленный media input на chunks через `ffmpeg`
- транскрибирует chunks параллельно
- собирает нормализованный `transcript.json` и `transcript.txt`
- опционально пишет `timestamps.txt`, `transcript.srt`, `transcript.vtt`, `diarized.json`, `raw.json`
- поддерживает OpenAI и Groq

## Приоритет конфигурации

Порядок приоритета:

1. `flags`
2. `WHISPER_CLI_*` в `env`
3. значения по умолчанию

Runtime `YAML`-конфиг больше не поддерживается. CLI использует GNU long flags вроде `--input`, `--provider` и `--output-dir`.

Если `--provider` и `--model` не заданы, CLI по умолчанию использует `provider=openai` и модель `gpt-4o-transcribe`.

## API keys

Для `provider=openai` основной Linux contract - Secret Service через `secret-tool`.

Сохранить ключ:

```bash
secret-tool store --label='whisper-cli OpenAI API key' application whisper-cli provider openai kind api-key
```

Проверить запись напрямую через Secret Service. Команда печатает ключ в `stdout`, `whisper-cli` при этом не запускается:

```bash
secret-tool lookup application whisper-cli provider openai kind api-key
```

Удалить ключ:

```bash
secret-tool clear application whisper-cli provider openai kind api-key
```

Если Secret Service недоступен или запись не найдена, CLI использует `OPENAI_API_KEY` как fallback. Этот fallback предназначен для CI и opt-in live smoke tests. Переменная должна быть экспортирована в окружение процесса: `echo $OPENAI_API_KEY` сам по себе не доказывает, что дочерний процесс её унаследует.

Для `provider=groq` текущий contract остаётся env-only: `GROQ_API_KEY`.

## Сборка и установка

```bash
make build
./bin/whisper-cli -h
```

```bash
make install
~/.local/bin/whisper-cli -h
```

`make build` пишет бинарник в `./bin/whisper-cli`.
`make install` создаёт `~/.local/bin` при необходимости и копирует туда бинарник.
Для проверки без записи в домашний каталог можно переопределить `INSTALL_DIR`, например `make install INSTALL_DIR=/tmp/whisper-cli-bin`.

## Bash completion

Разовый вариант без установки completion-файла:

```bash
make build
source <(./bin/whisper-cli completion bash)
```

Постоянная установка completion в Linux:

```bash
make install-bash-completion
```

`make install-bash-completion` создаёт `~/.local/share/bash-completion/completions` при необходимости и пишет туда `whisper-cli` completion script.
Для проверки без записи в домашний каталог можно переопределить `BASH_COMPLETION_DIR`, например `make install-bash-completion BASH_COMPLETION_DIR=/tmp/whisper-cli-completion`.
CLI-команда `whisper-cli completion bash` печатает script в `stdout` и не требует `API key` или `ffmpeg`.

## Быстрый старт

```bash
make build
```

```bash
./bin/whisper-cli \
  --input /path/to/media.mp4 \
  --output-dir /tmp/whisper-cli \
  --provider openai \
  --model whisper-1
```

```bash
GROQ_API_KEY=... ./bin/whisper-cli \
  --input /path/to/media-dir \
  --output-dir /tmp/whisper-cli \
  --provider groq \
  --model whisper-large-v3-turbo \
  --outputs timestamps,raw
```

Для моделей без `segment timestamps` CLI автоматически отключает `timestamps`. Для `gpt-4o-transcribe-diarize` это означает, что запуск без явного `--outputs` пишет `diarized.json` вместе с обязательными `transcript.json`/`transcript.txt`. Если нужны только plain output-артефакты без других optional artifacts, можно явно указать:

```bash
./bin/whisper-cli \
  --input /path/to/media.ogg \
  --provider openai \
  --model gpt-4o-transcribe \
  --outputs none
```

CLI распознаёт `flac`, `m4a`, `mp3`, `mp4`, `mpeg`, `mpga`, `ogg`, `wav`, `webm` как входные media extensions.
Любой поддерживаемый non-`m4a` input сначала конвертируется в `<output>/<base>/_work/source.m4a`, после чего chunking идёт уже по этому файлу.

## Флаги CLI

Дополнительная команда:

- `completion bash`

- `--provider`
- `--model`
- `--input`
- `--output-dir`
- `--language`
- `--outputs`
- `--chunk-seconds`
- `--concurrency`
- `--prompt`

`--outputs` управляет только optional artifacts. `transcript.json` и `transcript.txt` создаются всегда. Если модель не поддерживает `segment timestamps`, `timestamps` автоматически отключаются с warning.
Для `gpt-4o-transcribe-diarize` default `outputs=timestamps` после такого отключения заменяется на `diarized`, поэтому по умолчанию создаётся `diarized.json`. Явные `--outputs none`, `--outputs raw` или `WHISPER_CLI_OUTPUTS=...` уважаются без auto-добавления `diarized`.

Поддерживаемые optional outputs:

- `timestamps`
- `srt`
- `vtt`
- `diarized`
- `raw`
- `none`

## Матрица provider'ов

| Provider | Модель | Сегменты | SRT/VTT | Диаризация |
| --- | --- | --- | --- | --- |
| OpenAI | `whisper-1` | да | да | нет |
| OpenAI | `gpt-4o-transcribe` | нет | нет | нет |
| OpenAI | `gpt-4o-mini-transcribe` | нет | нет | нет |
| OpenAI | `gpt-4o-transcribe-diarize` | без subtitle-артефактов | нет | да |
| Groq | `whisper-large-v3` | да | да | нет |
| Groq | `whisper-large-v3-turbo` | да | да | нет |

OpenRouter пока не реализован: см. [`docs/ROADMAP.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/ROADMAP.md) и [`docs/tech-debt-tracker.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/tech-debt-tracker.md).

## Выходные артефакты

Для файла `lecture.mp4` CLI пишет в `<output-dir>/lecture/`:

- `transcript.json`
- `transcript.txt`
- `timestamps.txt` при `outputs=timestamps`
- `transcript.srt` при `outputs=srt`
- `transcript.vtt` при `outputs=vtt`
- `diarized.json` при `outputs=diarized`
- `raw.json` при `outputs=raw`
- `_work/source.m4a` для non-`m4a` input
- `_work/chunk_*.m4a` как промежуточные chunk-файлы

Если вход уже `lecture.m4a`, `_work/source.m4a` не создаётся.

## Проверки качества

Внешнего CI для этого проекта нет. Единственный официальный локальный `quality gate` сейчас `make ci`.

- `make fmt`
- `make fmt-check`
- `make lint`
- `make test`
- `make vet`
- `make docs-check`
- `make ci`

Опциональные `live smoke tests`:

- `make test-live-openai LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-openai-diarize LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-groq LIVE_AUDIO_FILE=/abs/path/audio.m4a`

## Документация

Точка входа для людей и агентов: [`docs/index.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/index.md)
