# Long-Run Harness Report

- Run ID: `demo-20260424-001555-860a01`
- Status: `completed`
- Started: `2026-04-24T00:15:55Z`
- Ended: `2026-04-24T00:15:55Z`
- Duration: `0s`
- Final step: `3`
- Resume count: `0`

## Objective

Create and verify a deterministic demo artifact in the workspace.

Requirements:
1. Write a file named demo.txt.
2. Read it back to confirm it exists and contains the expected text.
3. Stop once the evidence is sufficient.


## Final Answer

Completed a deterministic end-to-end run: wrote demo.txt, read it back, and persisted the run state, transcript, checkpoints, and report.

## Rolling Summary

Workspace is empty. Create a deterministic demo artifact first.
The file should now exist. Read it back as evidence.
The run is making steady progress and has real evidence in the workspace.
The artifact was created and verified.

## Sticky Facts

1. Workspace and artifacts are the durable source of truth.
2. Keep prompt context compact; summarize aggressively when needed.

## Current Plan

1. Create demo file
2. Inspect the file
3. Finish with a concise summary

## Current TODO

1. Prefer evidence gathering over speculative edits.
2. Read demo.txt
3. Return final answer

## Stats

- Steps: 3
- Actor calls: 3
- Reviewer calls: 1
- Summarizer calls: 0
- Tool calls: 2
- Tool failures: 0
- Reviews: 1
- Summaries: 0
- Checkpoints: 3

## Recent Events

- [2026-04-24T00:15:55Z] step=0 kind=objective actor=system
- [2026-04-24T00:15:55Z] step=1 kind=actor_decision actor=actor
- [2026-04-24T00:15:55Z] step=1 kind=tool_result actor=tool tool=files.write artifact=artifacts/step-0001-action-01-files-write.json
- [2026-04-24T00:15:55Z] step=2 kind=actor_decision actor=actor
- [2026-04-24T00:15:55Z] step=2 kind=tool_result actor=tool tool=files.read artifact=artifacts/step-0002-action-01-files-read.json
- [2026-04-24T00:15:55Z] step=2 kind=review actor=reviewer
- [2026-04-24T00:15:55Z] step=3 kind=actor_decision actor=actor

## Paths

- Run directory: `examples/runs/demo-20260424-001555-860a01`
- Workspace: `examples/workspace`
- State file: `examples/runs/demo-20260424-001555-860a01/state.json`
- Transcript: `examples/runs/demo-20260424-001555-860a01/transcript.jsonl`
