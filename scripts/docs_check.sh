#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "AGENTS.md"
  "README.md"
  "Makefile"
  "config.example.yaml"
  "docs/index.md"
  "docs/ARCHITECTURE.md"
  "docs/MAINTENANCE.md"
  "docs/PROJECT_REVIEW_WORKFLOW.md"
  "docs/ROADMAP.md"
  "docs/tech-debt-tracker.md"
  "docs/exec-plans/README.md"
  "docs/exec-plans/template.md"
  "docs/exec-plans/active/README.md"
  "docs/exec-plans/completed/README.md"
)

for file in "${required_files[@]}"; do
  test -f "$file" || {
    echo "missing required file: $file"
    exit 1
  }
done

grep -q '^# AGENTS Map' AGENTS.md
grep -q '^# Индекс документации' docs/index.md
grep -q '^# Архитектура' docs/ARCHITECTURE.md
grep -q '^# Эксплуатация' docs/MAINTENANCE.md
grep -q '^# Регламент ревью проекта' docs/PROJECT_REVIEW_WORKFLOW.md
grep -q '^# Дорожная карта' docs/ROADMAP.md
grep -q '^## Done$' docs/ROADMAP.md
grep -q '^## In Progress$' docs/ROADMAP.md
grep -q '^## Planned$' docs/ROADMAP.md
grep -q '^## Blocked$' docs/ROADMAP.md
grep -q 'make ci' AGENTS.md
grep -q 'make ci' README.md

for file in $(find docs/exec-plans/active -maxdepth 1 -type f ! -name README.md | sort); do
  grep -q '^Status:' "$file"
  grep -q '^## Цель$' "$file"
  grep -q '^## Проверка$' "$file"
done

for file in $(find docs/exec-plans/completed -maxdepth 1 -type f ! -name README.md | sort); do
  grep -q '^Status: Completed$' "$file"
  grep -q '^## Цель$' "$file"
  grep -q '^## Проверка$' "$file"
  grep -q '^## Журнал решений$' "$file"
done
