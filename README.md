# whisper-cli

Go CLI для локальной транскрипции медиафайлов и директорий через OpenAI-compatible speech-to-text providers.

## Что делает CLI

- нормализует поддерживаемые аудио- и видеоформаты в `m4a` через `ffmpeg` перед транскрипцией
- режет подготовленный media input на chunks через `ffmpeg`
- транскрибирует chunks параллельно
- последовательно обрабатывает несколько inputs через `whisper-cli --input *.m4a`
- по умолчанию пишет рядом с source только `<media-stem>.txt`
- хранит canonical `transcript.json`, converted source и chunks в системном cache
- опционально пишет stem-prefixed timestamps, subtitles и raw artifacts
- поддерживает OpenAI и Groq

## Приоритет конфигурации

Порядок приоритета:

1. `flags`
2. `WHISPER_CLI_*` в `env`
3. значения по умолчанию

Runtime `YAML`-конфиг больше не поддерживается. CLI использует GNU long flags вроде `--input`, `--provider` и `--output-dir`.

Если `--provider` и `--model` не заданы, CLI по умолчанию использует `provider=openai` и модель `gpt-transcribe`.

Для OpenAI поддерживается только `gpt-transcribe`. Старые явно заданные OpenAI model IDs не переписываются на default и завершают запуск ошибкой. Generic `--model` и `WHISPER_CLI_MODEL` остаются для моделей Groq и будущих providers.

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
  --provider openai \
  --model gpt-transcribe
```

Результат будет записан в `/path/to/media.txt`.

Последовательно обработать все `m4a` в каталоге:

```bash
./bin/whisper-cli --input /path/to/records/*.m4a
```

Wildcard раскрывает shell: `*.m4a` не нужно заключать в кавычки. Первый expanded path становится значением `--input`, остальные передаются CLI как продолжение input list.

```bash
GROQ_API_KEY=... ./bin/whisper-cli \
  --input /path/to/media-dir \
  --output-dir /tmp/whisper-cli \
  --provider groq \
  --model whisper-large-v3-turbo \
  --outputs timestamps,raw
```

`--output-dir` собирает public artifacts плоско в указанном каталоге. До первого provider request CLI проверяет, что разные inputs не претендуют на один output path.

Default `outputs=none`. Для моделей без `segment timestamps`, включая `gpt-transcribe`, явно запрошенный `timestamps` автоматически отключается с warning.

CLI распознаёт `flac`, `m4a`, `mp3`, `mp4`, `mpeg`, `mpga`, `ogg`, `wav`, `webm` как входные media extensions.
Любой поддерживаемый non-`m4a` input сначала конвертируется в системном cache, после чего chunking идёт уже по подготовленному файлу.

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

`--input` задаёт первый file или directory. Дополнительные paths после него образуют ordered batch; directory inputs по-прежнему сканируются non-recursively и сортируются по имени.

Без `--output-dir` public artifacts пишутся рядом с каждым source. Явный `--output-dir` или `WHISPER_CLI_OUTPUT_DIR` перенаправляет их в один каталог.

`--outputs` управляет только дополнительными public artifacts. Default — `none`; plain `<stem>.txt` и cached `transcript.json` создаются всегда. Если модель не поддерживает `segment timestamps`, `timestamps` автоматически отключается с warning.

Для `provider=openai` значение `--language` передаётся `gpt-transcribe` как один элемент `languages[]`. CLI сохраняет singular flag, чтобы не менять общий provider-independent интерфейс.

Поддерживаемые optional outputs:

- `timestamps`
- `srt`
- `vtt`
- `raw`
- `none`

## Матрица provider'ов

| Provider | Модель | Сегменты | SRT/VTT |
| --- | --- | --- | --- |
| OpenAI | `gpt-transcribe` | нет | нет |
| Groq | `whisper-large-v3` | да | да |
| Groq | `whisper-large-v3-turbo` | да | да |

OpenRouter пока не реализован: см. [`docs/ROADMAP.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/ROADMAP.md) и [`docs/tech-debt-tracker.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/tech-debt-tracker.md).

## Выходные артефакты

Для source `/records/lecture.mp4` default public output содержит только:

- `/records/lecture.txt`

При явном `--outputs timestamps,srt,vtt,raw` имена дополнительных public artifacts строятся от source stem:

- `lecture.timestamps.txt`
- `lecture.srt`
- `lecture.vtt`
- `lecture.raw.json`

Служебные данные последнего запуска хранятся в `<os-user-cache>/whisper-cli/<sha256-absolute-input>/`:

- `transcript.json`
- `work/source.m4a` для non-`m4a` input
- `work/chunk_*.m4a`

На Linux `<os-user-cache>` определяется через `os.UserCacheDir`: обычно это `$XDG_CACHE_HOME` или `~/.cache`. Перед повторной обработкой того же absolute input его cache directory заменяется, поэтому stale chunks не подмешиваются. Cache не используется для пропуска provider requests.

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
- `make test-live-groq LIVE_AUDIO_FILE=/abs/path/audio.m4a`

## Документация

Точка входа для людей и агентов: [`docs/index.md`](/home/arykalin/go/src/github.com/arykalin/whisper-cli/docs/index.md)
