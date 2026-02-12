---
topic: doctor-validation
plan: ../../planning/doctor-validation.md
format: local-markdown
status: in-progress
task_gate_mode: auto
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
current_phase: 2
current_task: doctor-validation-2-3
completed_phases: [1]
completed_tasks: [doctor-validation-1-1, doctor-validation-1-2, doctor-validation-1-3, doctor-validation-1-4, doctor-validation-2-1, doctor-validation-2-2]
started: 2026-02-12
updated: 2026-02-12
completed: ~
---

# Implementation: Doctor Validation

Implementation started.
