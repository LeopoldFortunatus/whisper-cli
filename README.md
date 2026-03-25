# whisper-cli

Go CLI для локальной транскрипции аудиофайлов и директорий через OpenAI-compatible speech-to-text providers.

## Что делает CLI

- режет длинные аудиофайлы на chunks через `ffmpeg`
- транскрибирует chunks параллельно
- собирает нормализованный `transcript.json` и `transcript.txt`
- опционально пишет `timestamps.txt`, `transcript.srt`, `transcript.vtt`, `diarized.json`, `raw.json`
- поддерживает OpenAI и Groq

## Приоритет конфигурации

Порядок приоритета:

1. `flags`
2. `WHISPER_CLI_*` в `env`
3. `config.yaml` как legacy-файл
4. значения по умолчанию

`config.yaml` не обязателен. Для примера есть [`config.example.yaml`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/config.example.yaml).

## Обязательные переменные окружения

- `OPENAI_API_KEY` для `provider=openai`
- `GROQ_API_KEY` для `provider=groq`

Важно: переменная должна быть экспортирована в окружение процесса. Команда `echo $OPENAI_API_KEY` сама по себе не доказывает это: shell-переменная видна `echo`, но дочерние процессы её не унаследуют без `export`.

```bash
export OPENAI_API_KEY=...
./bin/whisper-cli -h
```

Одноразовый вариант без `export`:

```bash
OPENAI_API_KEY=... ./bin/whisper-cli -h
```

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

## Быстрый старт

```bash
make build
```

```bash
OPENAI_API_KEY=... ./bin/whisper-cli \
  -input /path/to/audio.m4a \
  -output-dir /tmp/whisper-cli \
  -provider openai \
  -model whisper-1
```

```bash
GROQ_API_KEY=... ./bin/whisper-cli \
  -input /path/to/audio-dir \
  -output-dir /tmp/whisper-cli \
  -provider groq \
  -model whisper-large-v3-turbo \
  -outputs timestamps,raw
```

Для моделей без `segment timestamps` CLI автоматически отключает `timestamps` и оставляет обязательные `transcript.json`/`transcript.txt`. Если нужны только plain output-артефакты без других optional artifacts, можно явно указать:

```bash
OPENAI_API_KEY=... ./bin/whisper-cli \
  -input /path/to/audio.m4a \
  -provider openai \
  -model gpt-4o-transcribe \
  -outputs none
```

## Флаги CLI

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

`-outputs` управляет только optional artifacts. `transcript.json` и `transcript.txt` создаются всегда. Если модель не поддерживает `segment timestamps`, `timestamps` автоматически отключаются с warning.

Поддерживаемые optional outputs:

- `timestamps`
- `srt`
- `vtt`
- `diarized`
- `raw`
- `none`

## Совместимость с legacy-конфигом

Legacy-поля `input_file` и `usergpt4` всё ещё поддерживаются в YAML-конфиге:

- `input_file` маппится в `input`
- `usergpt4=true` маппится в `model=gpt-4o-transcribe`
- `usergpt4=false` маппится в `model=whisper-1`

При использовании legacy-полей CLI пишет `deprecation warnings`.

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

Для файла `lecture.m4a` CLI пишет в `<output-dir>/lecture/`:

- `transcript.json`
- `transcript.txt`
- `timestamps.txt` при `outputs=timestamps`
- `transcript.srt` при `outputs=srt`
- `transcript.vtt` при `outputs=vtt`
- `diarized.json` при `outputs=diarized`
- `raw.json` при `outputs=raw`

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
