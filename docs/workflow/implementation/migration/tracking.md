---
topic: migration
plan: ../../planning/migration.md
format: local-markdown
status: in-progress
task_gate_mode: auto
fix_gate_mode: gated
fix_attempts: 0
linters:
  - name: gofmt
    command: gofmt -l .
  - name: go-vet
    command: go vet ./...
  - name: golangci-lint
    command: ~/go/bin/golangci-lint run ./...
analysis_cycle: 1
project_skills:
  - name: golang-pro
    path: .claude/skills/golang-pro
current_phase: 3
current_task: migration-3-3
completed_phases:
  - 1
  - 2
completed_tasks:
  - migration-1-1
  - migration-1-2
  - migration-1-3
  - migration-1-4
  - migration-1-5
  - migration-2-1
  - migration-2-2
  - migration-2-3
  - migration-2-4
  - migration-2-5
  - migration-3-1
  - migration-3-2
started: 2026-02-15
updated: 2026-02-15
completed: ~
---

# Implementation: Migration

Implementation started.
