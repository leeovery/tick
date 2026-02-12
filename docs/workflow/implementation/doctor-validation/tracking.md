---
topic: doctor-validation
plan: ../../planning/doctor-validation.md
format: local-markdown
status: in-progress
task_gate_mode: gated
fix_gate_mode: gated
fix_attempts: 0
linters:
  - name: go-vet
    command: go vet ./...
  - name: gofmt
    command: gofmt -l .
  - name: golangci-lint
    command: ~/go/bin/golangci-lint run ./...
analysis_cycle: 0
project_skills: [golang-pro]
current_phase: 1
current_task: doctor-validation-1-2
completed_phases: []
completed_tasks: [doctor-validation-1-1]
started: 2026-02-12
updated: 2026-02-12
completed: ~
---

# Implementation: Doctor Validation

Implementation started.
