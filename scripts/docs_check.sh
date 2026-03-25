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
  "docs/ROADMAP.md"
  "docs/tech-debt-tracker.md"
  "docs/exec-plans/README.md"
  "docs/exec-plans/active/README.md"
  "docs/exec-plans/completed/README.md"
  "docs/exec-plans/completed/2026-03-25-agent-harness-multi-provider-refactor.md"
  ".github/workflows/ci.yml"
)

for file in "${required_files[@]}"; do
  test -f "$file" || {
    echo "missing required file: $file"
    exit 1
  }
done

grep -q '^# AGENTS Map' AGENTS.md
grep -q '^# Docs Index' docs/index.md
grep -q '^# Architecture' docs/ARCHITECTURE.md
grep -q '^# Maintenance' docs/MAINTENANCE.md
grep -q '^# Roadmap' docs/ROADMAP.md
grep -q 'make ci' AGENTS.md
grep -q 'make ci' README.md
