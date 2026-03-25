# Tech Debt Tracker

Owner: Platform Team
Last Verified: 2026-03-25

## Active

### TD-001 OpenRouter Transcription Contract Is Unclear
- Impact: нельзя безопасно объявить OpenRouter как supported transcription provider без drift относительно official API surface.
- Plan: при возврате к slice `RM-008` повторно проверить official docs/OpenAPI и либо реализовать adapter, либо явно оставить provider blocked.

## Closed

- none
