# P1 Implementation Plan — CCTV Health Monitor

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development

**Goal:** Implement 15+ tasks across Go backend and React/TypeScript frontend

**Architecture:** Compliance-first, modular code, clean architecture

**Tech Stack:** Go 1.25 (Chi, pgx/v5, NATS) + React 19 (Vite 8, Zustand, React Flow, FullCalendar) + TypeScript 5.9

---

## Task Groups (Independent — parallel execution)

### Group A: Backend Core Tests & Fixes
- BACKEND.1: ActionExecutor Unit Tests
- BACKEND.3: CMMSIntegrator Context Timeouts
- PERF.4: Health Checks Enhancement
- ARCH.4: Replace http.Error with respondError
- ARCH.5: Trace ID Propagation

### Group B: Playbook & RCA
- BACKEND.2: PlaybookRegistry Versioning
- BACKEND.4: RCA Graph Auto-Update
- BACKEND.5: RCA BuildFromState Accuracy

### Group C: Frontend Architecture
- ARCH.1: Context Migration to Zustand
- ARCH.2: API Routes Organization
- ARCH.3: OpenAPI TypeScript Generation

### Group D: Frontend Features
- WF.1: Workflow Builder UI Enhancement
- WF.2: Resource Planning Calendar Enhancement
- INT.1: Webhook Builder UI Enhancement
- PERF.5: Redis for SLA Trackers
