# TD-003 Targets для build/install

Владелец: Platform Team
Проверено: 2026-03-25
Status: Completed

## Цель

Добавить воспроизводимые `make build` и `make install` для локальной сборки и установки CLI без дрейфа по имени бинарника, путям и документации.

## Контекст

- В `Makefile` есть `quality gates` и `opt-in live tests`, но нет стандартного локального пути сборки и установки бинарника.
- `README.md` и `docs/MAINTENANCE.md` не фиксируют, где должен появляться локальный бинарник и как проверять install без ручных шагов.
- Новый `build output` не должен попадать в `git-tracked state`.

## Риски

1. `Build target` может писать в отслеживаемый путь и засорять рабочее дерево артефактами.
2. `Install target`, жёстко привязанный к `~/.local/bin`, неудобно валидировать автоматически и можно случайно затронуть пользовательское окружение.
3. Документация может описать `contract` иначе, чем его реализует `Makefile`.

## План

1. Зафиксировать `exec-plan` и выбрать единый `contract` для имени бинарника, `build path` и `install path`.
2. Добавить `build` и `install` `targets` и исключить `build output` из `git-tracked state`.
3. Синхронизировать `README.md` и `docs/MAINTENANCE.md` с фактическим `contract`.
4. Прогнать `make build`, `make install` с переопределённым `INSTALL_DIR`, `make ci`, затем закрыть долг и перенести план в `completed/`.

## Проверка

- `make build`
- `make install INSTALL_DIR=/tmp/whisper-cli-install`
- `make ci`

## Журнал решений

- 2026-03-25: по умолчанию складывать локальный `build artifact` в `./bin/whisper-cli`, чтобы `install` копировал детерминированный файл, а не собирал в произвольный путь.
- 2026-03-25: поддержать `override` через `INSTALL_DIR`, чтобы валидировать `install target` без записи в пользовательский `~/.local/bin`.
- 2026-03-25: `install` должен зависеть от `build`, чтобы пользователь получал ровно тот же бинарник и локально, и при установке.

## Находки

- `.gitignore` не исключал `bin/`, поэтому новый локальный `build output` нужно явно игнорировать.
- Проверка через `INSTALL_DIR=/tmp/whisper-cli-install` подтверждает, что `install contract` можно валидировать без побочного эффекта в домашнем каталоге.

## Следующие шаги

- Если появится отдельный `packaging/release slice`, его лучше оформить новыми `targets` поверх локального `build/install`, а не перегружать текущий `contract`.

## Итоги

- В `Makefile` появились воспроизводимые `make build` и `make install` с единым `contract` для имени бинарника и путей.
- Локальный build output вынесен в игнорируемый `bin/`, поэтому рабочее дерево не загрязняется tracked-артефактами.
- `README.md`, `docs/MAINTENANCE.md` и `docs/tech-debt-tracker.md` синхронизированы с фактическим поведением targets.
- Проверка завершена: `make build`, `make install INSTALL_DIR=/tmp/whisper-cli-install`, `make ci`.
