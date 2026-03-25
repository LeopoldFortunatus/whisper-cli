# Tech Debt Tracker

Owner: Platform Team
Last Verified: 2026-03-25

## Active

### TD-001 OpenRouter Transcription Contract Is Unclear
- Impact: нельзя безопасно объявить OpenRouter как supported transcription provider без drift относительно official API surface.
- Plan: при возврате к slice `RM-008` повторно проверить official docs/OpenAPI и либо реализовать adapter, либо явно оставить provider blocked.

### TD-002 настроить линтер
- Impact: Сейчас нет линтер проверки
- Plan: Скопировать настройки и запуск из репо /home/arykalin/go/src/gitlab-private.wildberries.ru/cloud/rbac-interceptor
## Closed

- none
