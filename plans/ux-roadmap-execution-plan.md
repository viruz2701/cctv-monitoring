# UX Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development.
> Start with Top-5 Must-Do First, then continue with remaining tracks.

**Goal:** Execute 29 UX tasks across 6 tracks to transform CCTV Health Monitor UX

**Architecture:** Strangler Fig pattern — new components alongside old, Feature Flags for gradual rollout, route aliasing for backward compatibility

**Tech Stack:** React 19 + TypeScript 5.9 + TailwindCSS v4 + Zustand + React Query + Vite 8

---

## 🎯 Top-5 Must-Do First

### Task 1: UX-1.2 Unified Work Hub (3d)
**Feature Flag:** `unified_work_hub_v2`

**Files:**
- Create: `frontend/src/pages/UnifiedWorkHub.tsx`
- Create: `frontend/src/components/work-hub/WorkHubTabs.tsx`
- Create: `frontend/src/components/work-hub/QuickFilters.tsx`
- Create: `frontend/src/routes/workHubRoutes.ts`
- Create: `frontend/src/config/featureFlags.ts` (add flag)
- Modify: `frontend/src/AppShell.tsx` (add route)

**Acceptance:**
- Tab-based: [My Tasks] [Team] [Requests]
- Quick Filters: Overdue, Critical, Unassigned
- Route aliasing: /work-orders → /hub?tab=tasks, /tickets → /hub?tab=requests
- Bulk actions toolbar works in all tabs
- URL sharing preserves tab state
- npx tsc --noEmit = PASS

### Task 2: UX-3.2 Auto-fill TO Journals (3d)
**Feature Flag:** `to_auto_generation`

**Files:**
- Create: `frontend/src/components/work-orders/WOCompletionFlow.tsx`
- Modify: `frontend/src/pages/WorkOrderDetail/WorkOrderDetail.tsx`
- Create: `backend/internal/api/to_journal_handlers.go`
- Create: `backend/internal/compliance/to_journal.go`

**Acceptance:**
- Auto-creates journal entries on WO completion
- Pre-fills: device, date, technician, location, time
- Required fields marked as "manual" with ⚠
- go build ./... + npx tsc --noEmit = PASS

### Task 3: UX-4.2 QR Mobile Flow (4d)
**Feature Flag:** `mobile_qr_lifecycle`

**Files:**
- Create: `mobile/src/screens/DeviceOnboarding.tsx`
- Create: `mobile/src/services/qrLifecycle.ts`
- Modify: `mobile/src/screens/QRScannerScreen.tsx`
- Create: `backend/internal/api/qr_handlers.go`
- Create: `frontend/src/components/qr/QRPrintBatch.tsx`

**Acceptance:**
- Scan QR → Open WO → Fill checklist → Sign → Generate TO
- Offline mode with differential sync
- GPS verification on scan
- npx tsc --noEmit = PASS

### Task 4: UX-2.4 Secure Tunnel (3d)
**Files:**
- Create: `frontend/src/components/device/SecureTunnel.tsx`
- Modify: `frontend/src/pages/DeviceDetail/DeviceDetail.tsx`
- Create: `frontend/src/services/tunnelApi.ts`

**Acceptance:**
- SSH/HTTPS proxy via WebSocket
- One-time token with TTL 1h
- Copy-to-clipboard tunnel URL
- QR code for mobile access
- Audit log on connection
- npx tsc --noEmit = PASS

### Task 5: UX-4.4 Schedule Builder (4d)
**Files:**
- Create: `frontend/src/components/schedule/ScheduleBuilder.tsx`
- Create: `frontend/src/components/schedule/RuleEditor.tsx`
- Modify: `frontend/src/pages/MaintenanceCalendar.tsx`

**Acceptance:**
- Select devices by site/type/vendor
- Apply regulatory template by region
- Assign technicians
- Review conflicts
- Generate schedule
- npx tsc --noEmit = PASS

---

## 📋 Track 1: Navigation & IA (W1-2)

### Task 6: UX-1.1 Sidebar Progressive Disclosure (3d)
### Task 7: UX-1.3 Route Aliasing Middleware (1d)
### Task 8: UX-1.4 Breadcrumbs Enhancement (1d)
### Task 9: UX-1.5 Role-Based Home Pages (2d)
### Task 10: UX-1.6 Sidebar A11y (1d)

## 📋 Track 2: Device Operations (W3-4)

### Task 11: UX-2.1 Three-Column Layout (2d)
### Task 12: UX-2.2 Device Live View (3d)
### Task 13: UX-2.3 Alert Center (3d)
### Task 14: UX-2.5 Device History Timeline (2d)

## 📋 Track 3: TO Compliance (W5-6)

### Task 15: UX-3.1 TO Journals with Templates (3d)
### Task 16: UX-3.3 TO Document Preview (2d)
### Task 17: UX-3.4 AI Copilot (3d)
### Task 18: UX-3.5 Print Template Editor (4d)
### Task 19: UX-3.6 Hash-Chain Signatures (3d)
### Task 20: UX-3.7 Regulatory Checklist (2d)

## 📋 Track 4: Mobile & Calendar (W7-8)

### Task 21: UX-4.1 Asset Tree Drill-down (4d)
### Task 22: UX-4.3 Maintenance Calendar UI (3d)

## 📋 Track 5: Command Palette (W3)

### Task 23: UX-5.1 Command Palette Regulatory (3d)

## 📋 Track 6: Performance & A11y (Ongoing)

### Task 24: UX-7.1 Bundle Size Optimization (2d)
### Task 25: UX-7.2 Image Optimization (2d)
### Task 26: UX-8.1 A11y Audit & CI Gate (2d)
### Task 27: UX-8.2 Keyboard Navigation Audit (2d)

## 📋 Cross-Cutting

### Task 28: Feature Flag Registry Setup (1d)
### Task 29: Storybook Stories for New Components (ongoing)
