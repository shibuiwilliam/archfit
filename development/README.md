# development/ — Technical Documentation for archfit Development

This directory contains detailed technical documentation to support ongoing development and enhancement of archfit. These documents complement `CLAUDE.md` (the agent contract) and `PROJECT.md` (the project overview).

## Document Index

### Core Subsystems

| Document | Purpose |
|---|---|
| [architecture.md](./architecture.md) | System architecture, package boundaries, data flow, and extension points |
| [api-reference.md](./api-reference.md) | Core types, interfaces, and function signatures |
| [metrics-and-scoring.md](./metrics-and-scoring.md) | How scores and metrics are computed and the normalization algorithm |
| [testing-strategy.md](./testing-strategy.md) | Three-layer testing approach, fixture conventions, and test helpers |
| [fix-engine.md](./fix-engine.md) | Fix engine internals: fixer interface, plan-apply-verify loop |
| [llm-integration.md](./llm-integration.md) | Multi-provider LLM architecture, prompt design, budget/cache system |
| [pack-development.md](./pack-development.md) | How to create, test, validate, and publish rule packs |
| [ci-cd-integration.md](./ci-cd-integration.md) | GitHub Action, SARIF, PR gate workflows, trend tracking |
| [cross-stack-improvements.md](./cross-stack-improvements.md) | Detection pattern expansion for Java, Ruby, PHP, Terraform, monorepos |

### Three Strategic Elements

These documents describe the three capabilities that transform archfit from a point-in-time audit tool into continuous architecture infrastructure.

| Document | Element | Status |
|---|---|---|
| [fitness-contract.md](./fitness-contract.md) | Fitness Contract as Code — hard constraints, area budgets, agent directives | Implemented (types, check logic, CLI) |
| [agent-observatory.md](./agent-observatory.md) | Agent Behavior Observatory — trace ingestion and behavioral metrics | Not started |
| [adaptive-engine.md](./adaptive-engine.md) | Adaptive Rule Engine — learning feedback loop for confidence and thresholds | Not started |

### Planning

| Document | Purpose |
|---|---|
| [implementation-plan.md](./implementation-plan.md) | Phased implementation plan with PR-sized work units |

## How to Use These Documents

**Claude Code**: reference these documents when working on specific subsystems. The `CLAUDE.md` contract takes precedence if there is a conflict.

**Human contributors**: read `architecture.md` first for the big picture, then the relevant subsystem document for your task.

**Adding a new document**: keep documents focused on one subsystem. Link from this README. Do not duplicate content from `CLAUDE.md` or `docs/`.
