# whisper-cli

Go CLI для локальной транскрипции аудиофайлов и директорий через OpenAI-compatible speech-to-text providers.

## What It Does

- режет длинные аудиофайлы на chunks через `ffmpeg`
- транскрибирует chunks параллельно
- собирает нормализованный `transcript.json` и `transcript.txt`
- опционально пишет `timestamps.txt`, `transcript.srt`, `transcript.vtt`, `diarized.json`, `raw.json`
- поддерживает OpenAI и Groq

## Configuration Priority

Порядок precedence:

1. flags
2. `WHISPER_CLI_*` env
3. `config.yaml` legacy file
4. defaults

`config.yaml` не обязателен. Для примера есть [`config.example.yaml`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/config.example.yaml).

## Required Environment

- `OPENAI_API_KEY` для `provider=openai`
- `GROQ_API_KEY` для `provider=groq`

## Quick Start

```bash
make test
```

```bash
OPENAI_API_KEY=... go run . \
  -input /path/to/audio.m4a \
  -output-dir /tmp/whisper-cli \
  -provider openai \
  -model whisper-1
```

```bash
GROQ_API_KEY=... go run . \
  -input /path/to/audio-dir \
  -output-dir /tmp/whisper-cli \
  -provider groq \
  -model whisper-large-v3-turbo \
  -outputs timestamps,raw
```

Если нужен только plain transcript для моделей без timestamps:

```bash
OPENAI_API_KEY=... go run . \
  -input /path/to/audio.m4a \
  -provider openai \
  -model gpt-4o-transcribe \
  -outputs none
```

## CLI Flags

- `-config`
- `-provider`
- `-model`
- `-input`
- `-output-dir`
- `-language`
- `-outputs`
- `-chunk-seconds`
- `-concurrency`
- `-prompt`

`-outputs` управляет только optional artifacts. `transcript.json` и `transcript.txt` создаются всегда.

Поддерживаемые optional outputs:

- `timestamps`
- `srt`
- `vtt`
- `diarized`
- `raw`
- `none`

## Legacy Compatibility

Legacy-поля `input_file` и `usergpt4` всё ещё поддерживаются в YAML-конфиге:

- `input_file` маппится в `input`
- `usergpt4=true` маппится в `model=gpt-4o-transcribe`
- `usergpt4=false` маппится в `model=whisper-1`

При использовании legacy-полей CLI пишет deprecation warnings.

## Provider Matrix

| Provider | Model | Segments | SRT/VTT | Diarization |
| --- | --- | --- | --- | --- |
| OpenAI | `whisper-1` | yes | yes | no |
| OpenAI | `gpt-4o-transcribe` | no | no | no |
| OpenAI | `gpt-4o-mini-transcribe` | no | no | no |
| OpenAI | `gpt-4o-transcribe-diarize` | no subtitles | no | yes |
| Groq | `whisper-large-v3` | yes | yes | no |
| Groq | `whisper-large-v3-turbo` | yes | yes | no |

OpenRouter пока не реализован: см. [`docs/ROADMAP.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/ROADMAP.md) и [`docs/tech-debt-tracker.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/tech-debt-tracker.md).

## Outputs

Для файла `lecture.m4a` CLI пишет в `<output-dir>/lecture/`:

- `transcript.json`
- `transcript.txt`
- `timestamps.txt` при `outputs=timestamps`
- `transcript.srt` при `outputs=srt`
- `transcript.vtt` при `outputs=vtt`
- `diarized.json` при `outputs=diarized`
- `raw.json` при `outputs=raw`

## Quality Loop

- `make fmt`
- `make fmt-check`
- `make test`
- `make vet`
- `make docs-check`
- `make ci`

Опциональные live smoke tests:

- `make test-live-openai LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-openai-diarize LIVE_AUDIO_FILE=/abs/path/audio.m4a`
- `make test-live-groq LIVE_AUDIO_FILE=/abs/path/audio.m4a`

## Docs

Точка входа для людей и агентов: [`docs/index.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/index.md)
