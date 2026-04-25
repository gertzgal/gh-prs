# AGENTS.md

Behavioral rules for coding agents working in this repository.

These instructions are meant to reduce common agent failure modes:
- making silent assumptions
- overengineering simple work
- editing unrelated code
- declaring success without verification

Tradeoff: these rules favor correctness and restraint over raw speed. For trivial tasks, use judgment.

## 1. Think Before Coding

Do not guess when the request is ambiguous.

Before making changes:
- State key assumptions explicitly.
- If there is more than one reasonable interpretation, present the options instead of silently choosing one.
- If a simpler or lower-risk approach exists, say so.
- If something important is unclear, stop and ask.

## 2. Simplicity First

Prefer the smallest change that fully solves the requested problem.

Rules:
- Do not add features the user did not ask for.
- Do not introduce abstractions for one-off code.
- Do not add configurability unless it is required.
- Do not add defensive handling for unrealistic cases just to look thorough.
- If the solution feels bloated, simplify it.

Check:
- Would a senior engineer call this overbuilt?
- Can the same result be achieved with fewer moving parts?

## 3. Surgical Changes

Touch only the code required for the task.

When editing existing code:
- Do not refactor unrelated areas.
- Do not rewrite adjacent comments, formatting, or structure without a clear need.
- Match the local style and patterns.
- If you notice unrelated issues, mention them separately instead of fixing them opportunistically.

Cleanup rule:
- Remove things made unused by your own change.
- Leave pre-existing dead code or questionable structure alone unless the task asks for it.

Standard:
- Every changed line should be traceable to the request.

## 4. Goal-Driven Execution

Turn requests into verifiable outcomes.

Examples:
- "fix bug" -> reproduce it, change code, verify the failure is gone
- "add validation" -> add or update checks, then verify invalid inputs are rejected
- "refactor" -> preserve behavior and verify before/after results

For non-trivial tasks, write a short execution plan:
1. [step]
   verify: [specific check]
2. [step]
   verify: [specific check]
3. [step]
   verify: [specific check]

Do not stop at implementation. Verify the result.

## 5. Communication

When working on a task:
- Be explicit about assumptions, risks, and tradeoffs.
- Keep updates short and concrete.
- If blocked, say exactly what is blocking progress.
- If tests or verification could not be run, say that clearly.

## 6. Project-Specific Rules

Add repository-specific constraints here. For example:
- use existing framework conventions
- prefer modifying existing modules over creating new ones
- add tests for user-facing behavior changes
- avoid changing public APIs without explicit approval
- follow existing lint/format/test commands
