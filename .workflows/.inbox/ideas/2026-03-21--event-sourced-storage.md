# Event-Sourced Storage

Replace the current JSONL storage model with an append-only mutation log where task state is derived from replaying events. Instead of updating lines in place, every change (create, update field, transition status, add dependency) would be appended as a discrete event. Current task state would be computed by replaying the log from the beginning.

The main motivation is multi-user support. The current JSONL model works well for single-user but breaks down with concurrent editors — updating an existing line creates merge conflicts and race conditions even with file locking. An append-only log sidesteps this entirely since writers never modify existing data, only append new entries.

This is a significant architectural change. The current system treats the JSONL file as a mutable source of truth with a SQLite cache for fast queries. An event-sourced model would shift the JSONL to an immutable event log, with the SQLite cache becoming the materialised view — conceptually similar to the current setup but with different write semantics.

Worth considering only if multi-user becomes a real goal. The current model is simple, fast, and well-suited to Tick's single-user design.
