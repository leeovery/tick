---
topic: migration
plan: ../../planning/migration.md
format: local-markdown
status: in-progress
task_gate_mode: gated
fix_gate_mode: gated
fix_attempts: 0
linters:
  - name: gofmt
    command: gofmt -l .
  - name: go-vet
    command: go vet ./...
  - name: golangci-lint
    command: ~/go/bin/golangci-lint run ./...
analysis_cycle: 0
project_skills:
  - name: golang-pro
    path: .claude/skills/golang-pro
current_phase: 1
current_task: migration-1-4
completed_phases: []
completed_tasks:
  - migration-1-1
  - migration-1-2
  - migration-1-3
started: 2026-02-15
updated: 2026-02-15
completed: ~
---

# Implementation: Migration

Implementation started.
