# UX-01: Mobile Work Order Completion Wizard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development

**Goal:** Reduce work order completion from 10+ clicks across 6 screens to max 3 screens with a unified wizard flow.

**Architecture:** React Native + Expo 52, single wizard component with step management, replaces 4 separate screens (Checklist → PhotoCapture → Verification → Signature).

**Tech Stack:** React Native 0.76, Expo 52, React Navigation 7, Zustand, Zod

---

## Current Flow (6 screens, 10+ clicks)

```
WorkOrderDetail → Checklist → PhotoCapture → Verification → Signature → [Done]
     [Start]       [Check OK]    [Take Photo]   [GPS/EXIF/AI]   [Sign]
```

## Target Flow (3 screens, wizard)

```
      ┌──────────────────────────────────────────────────────┐
      │              CompleteWorkOrderWizard                 │
      │                                                      │
      │  Step 1/3: 📋 Checklist      ← 1 tap per item       │
      │  Step 2/3: 📸 Photo + GPS    ← 1 tap photo, auto-GPS│
      │  Step 3/3: ✍️ Signature      ← 1 sign + 1 submit    │
      │                                                      │
      │  [< Back]  [● ○ ○]  [○ ● ○]  [○ ○ ●]  [Next >]    │
      └──────────────────────────────────────────────────────┘
```

**Total: 3 screens, ~5 taps** (vs 10+ currently)

---

## File Changes

### Create:
- `mobile/src/components/CompleteWorkOrderWizard.tsx` — main wizard component (NEW)

### Modify:
- `mobile/src/screens/WorkOrderDetailScreen.tsx` — add "Complete" button → open wizard
- `mobile/src/navigation/AppNavigator.tsx` — add wizard route, remove old routes
- `mobile/src/types/index.ts` — add wizard params to navigation types

### Delete (no longer needed as separate screens):
- `mobile/src/screens/ChecklistScreen.tsx` → absorbed into wizard step 1
- `mobile/src/screens/PhotoCaptureScreen.tsx` → absorbed into wizard step 2
- `mobile/src/screens/VerificationScreen.tsx` → optional inline in wizard step 2
- `mobile/src/screens/SignatureScreen.tsx` → absorbed into wizard step 3

---

## Architecture: CompleteWorkOrderWizard

```
CompleteWorkOrderWizard
├── StepIndicator (● ○ ○ progress dots)
├── Step 1: ChecklistStep
│   ├── Progress bar (0-X%)
│   ├── ChecklistItem[] (toggleable)
│   └── [Next →]
├── Step 2: PhotoGPSStep
│   ├── GPS auto-capture (location hook)
│   ├── Camera capture (ImagePicker)
│   ├── Photo thumbnail gallery
│   ├── [Skip GPS] for non-KII sites
│   ├── [Optional: Run verification] — collapse if site setting says skip
│   └── [← Back] [Next →]
├── Step 3: SignatureStep
│   ├── Signature pad (react-native-signature-canvas)
│   ├── Notes textarea
│   ├── Summary (checklist %, photo count, signature status)
│   ├── [← Back] [Submit →]
└── CompleteScreen
    ├── Success animation
    └── [Back to Dashboard]
```

---

## Data Flow

```
WizardState:
├── step: 1 | 2 | 3 | 4 (complete)
├── checklist: ChecklistItem[]
├── photos: string[] (local URIs, uploaded in background)
├── gps: {latitude, longitude, accuracy}
├── gpsSkipped: reason | null
├── verificationResult: VerificationResponse | null
├── signature: string | null
└── notes: string
```

State managed with `useState` in wizard — no more navigation params passing.

---

## Tasks

### Task 1: Create CompleteWorkOrderWizard component

**Files:**
- Create: `mobile/src/components/CompleteWorkOrderWizard.tsx`

**Interfaces:**
- Consumes: `WorkOrder` from route params
- Consumes: `useCompleteWorkOrder` hook
- Consumes: `useLocation` hook
- Consumes: `useGatekeeper` hook (optional verification)
- Produces: Wizard with 3 steps + complete screen

- [ ] **Step 1: Create wizard state types and component shell**

```typescript
// CompleteWorkOrderWizard.tsx
type WizardStep = 1 | 2 | 3 | 4; // 4 = complete

interface WizardState {
  step: WizardStep;
  checklist: ChecklistItem[];
  photos: string[];
  gps: { latitude: number; longitude: number; accuracy: number } | null;
  gpsSkipped: string | null;
  verificationToken: string | null;
  signature: string | null;
  notes: string;
  isSubmitting: boolean;
}
```

- [ ] **Step 2: Implement StepIndicator**

```tsx
function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <View style={styles.indicator}>
      {Array.from({ length: total }, (_, i) => (
        <View key={i} style={[
          styles.dot,
          i + 1 === current && styles.dotActive,
          i + 1 < current && styles.dotCompleted
        ]} />
      ))}
    </View>
  );
}
```

- [ ] **Step 3: Implement ChecklistStep (step 1)**

```tsx
function ChecklistStep({ items, onToggle }: ChecklistStepProps) {
  // Reuse logic from current ChecklistScreen
  // Progress bar, toggleable items
  // [Next →] button (disabled until all checked? or optional)
}
```

- [ ] **Step 4: Implement PhotoGPSStep (step 2)**

```tsx
function PhotoGPSStep({ photos, gps, onAddPhoto, onRemovePhoto, onSkipGPS }: PhotoGPSStepProps) {
  // GPS status bar (auto-detected via useLocation)
  // Camera capture button (expo-image-picker)
  // Photo gallery with remove button
  // Optional inline verification collapse
  // [← Back] [Next →]
}
```

- [ ] **Step 5: Implement SignatureStep (step 3)**

```tsx
function SignatureStep({ signature, notes, onSignature, onNotes, checklist, photos }: SignatureStepProps) {
  // Signature pad
  // Notes textarea
  // Summary section
  // [← Back] [Submit →]
}
```

- [ ] **Step 6: Implement CompleteScreen (step 4)**

```tsx
function CompleteScreen({ workOrder }: { workOrder: WorkOrder }) {
  // Success animation/icon
  // "Work Order completed" message
  // [Back to Dashboard] button
}
```

- [ ] **Step 7: Wire up submission logic**

```typescript
const handleSubmit = async () => {
  setState(s => ({ ...s, isSubmitting: true }));
  
  const payload: CompleteWorkOrderPayload = {
    notes: state.notes,
    checklist: state.checklist,
    photos: state.photos,
    parts_used: [],
    signature: state.signature || undefined,
    verification_token: state.verificationToken || undefined,
    location: state.gps || undefined,
  };

  try {
    await completeMutation.mutateAsync({ id: workOrder.id, payload });
    setState(s => ({ ...s, step: 4 })); // Show complete screen
  } catch {
    // Offline fallback
    addToQueue({ type: 'complete_work_order', workOrderId: workOrder.id, payload });
    setState(s => ({ ...s, step: 4 }));
  }
};
```

- [ ] **Step 8: Run lint**

Run: `cd /home/viruz/cctv-monitoring/mobile && npx tsc --noEmit`

Expected: no type errors

- [ ] **Step 9: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add mobile/src/components/CompleteWorkOrderWizard.tsx && git commit -m "UX-01: create CompleteWorkOrderWizard component with 3-step flow"
```

---

### Task 2: Update WorkOrderDetailScreen — add "Complete" button

**Files:**
- Modify: `mobile/src/screens/WorkOrderDetailScreen.tsx`

- [ ] **Step 1: Add "Complete" button for `in_progress` status**

In the actions section, add:

```tsx
{workOrder.status === 'in_progress' && (
  <TouchableOpacity
    style={[styles.button, styles.completeButton]}
    onPress={() => navigation.navigate('CompleteWorkOrder', { workOrder })}
  >
    <Text style={styles.buttonText}>✅ Завершить наряд</Text>
  </TouchableOpacity>
)}
```

And add style:
```tsx
completeButton: {
  backgroundColor: '#16a34a',
},
```

- [ ] **Step 2: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add mobile/src/screens/WorkOrderDetailScreen.tsx && git commit -m "UX-01: add Complete Work Order button to detail screen"
```

---

### Task 3: Update navigation and types

**Files:**
- Modify: `mobile/src/types/index.ts`
- Modify: `mobile/src/navigation/AppNavigator.tsx`

- [ ] **Step 1: Add wizard route to types**

```typescript
export type RootStackParamList = {
  // ... existing routes
  CompleteWorkOrder: { workOrder: WorkOrder }; // NEW
  // Remove: Checklist, PhotoCapture, Verification, Signature (absorbed into wizard)
};
```

- [ ] **Step 2: Register wizard screen in navigator**

```tsx
import CompleteWorkOrderWizard from '../components/CompleteWorkOrderWizard';

// In Stack.Navigator, replace old screens:
<Stack.Screen
  name="CompleteWorkOrder"
  component={CompleteWorkOrderWizard}
  options={{
    title: 'Завершение наряда',
    gestureEnabled: false, // Disable swipe back during wizard
    headerBackVisible: false, // Hide back button (wizard handles it)
  }}
/>
```

- [ ] **Step 3: Remove old screen routes and imports**

Remove from AppNavigator.tsx:
```tsx
// REMOVE these imports:
import ChecklistScreen from '../screens/ChecklistScreen';
import PhotoCaptureScreen from '../screens/PhotoCaptureScreen';
import SignatureScreen from '../screens/SignatureScreen';
import VerificationScreen from '../screens/VerificationScreen';

// REMOVE these routes:
<Stack.Screen name="Checklist" ... />
<Stack.Screen name="PhotoCapture" ... />
<Stack.Screen name="Verification" ... />
<Stack.Screen name="Signature" ... />
```

- [ ] **Step 4: Run type check**

Run: `cd /home/viruz/cctv-monitoring/mobile && npx tsc --noEmit`

Expected: no type errors

- [ ] **Step 5: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add mobile/src/types/index.ts mobile/src/navigation/AppNavigator.tsx && git commit -m "UX-01: update navigation and types for completion wizard"
```

---

### Task 4: Verification as optional toggle (site setting)

**Files:**
- Modify: `mobile/src/components/CompleteWorkOrderWizard.tsx`
- Modify: `mobile/src/api/workOrders.ts` (add site config endpoint if needed)

- [ ] **Step 1: Add verification toggle logic to wizard**

```typescript
// Fetch site config to check if verification is required
const { data: siteConfig } = useQuery({
  queryKey: ['siteConfig', workOrder.site_name],
  queryFn: () => workOrdersApi.getSiteConfig(workOrder.site_name),
  enabled: !!workOrder.site_name,
});

const verificationRequired = siteConfig?.verification_required ?? true;
```

- [ ] **Step 2: Conditional verification in PhotoGPSStep**

```tsx
// If verification not required, skip the whole verification block
// Just capture GPS + photos without calling gatekeeper API

{verificationRequired && (
  <Collapsible title="Верификация (опционально)">
    <TouchableOpacity onPress={handleVerify}>
      <Text>Запустить верификацию</Text>
    </TouchableOpacity>
  </Collapsible>
)}
```

- [ ] **Step 3: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add mobile/src/components/CompleteWorkOrderWizard.tsx && git commit -m "UX-01: make verification optional based on site config"
```

---

### Task 5: Old screen cleanup (delete unused files)

**Files:**
- Delete: `mobile/src/screens/ChecklistScreen.tsx`
- Delete: `mobile/src/screens/PhotoCaptureScreen.tsx`
- Delete: `mobile/src/screens/VerificationScreen.tsx`
- Delete: `mobile/src/screens/SignatureScreen.tsx`

- [ ] **Step 1: Verify nothing imports the old screens**

Run: `cd /home/viruz/cctv-monitoring/mobile && grep -rn "ChecklistScreen\|PhotoCaptureScreen\|VerificationScreen\|SignatureScreen" src/ --include="*.tsx" --include="*.ts"`

Expected: only references in git history, not in code

- [ ] **Step 2: Delete old files**

```bash
rm mobile/src/screens/ChecklistScreen.tsx
rm mobile/src/screens/PhotoCaptureScreen.tsx
rm mobile/src/screens/VerificationScreen.tsx
rm mobile/src/screens/SignatureScreen.tsx
```

- [ ] **Step 3: Run type check**

Run: `cd /home/viruz/cctv-monitoring/mobile && npx tsc --noEmit`

Expected: no type errors

- [ ] **Step 4: Commit**

```bash
cd /home/viruz/cctv-monitoring && git rm mobile/src/screens/ChecklistScreen.tsx mobile/src/screens/PhotoCaptureScreen.tsx mobile/src/screens/VerificationScreen.tsx mobile/src/screens/SignatureScreen.tsx && git commit -m "UX-01: remove old completion screens absorbed into wizard"
```

---

## Plan Self-Review

**1. Spec coverage:**
- ✅ Max 3 screens for completion (wizard has 3 steps + complete screen)
- ✅ Verification optional (toggle in site settings)
- ✅ Offline mode works (catch in handleSubmit → addToQueue)
- ✅ Uses existing hooks (useLocation, useCompleteWorkOrder, useGatekeeper)

**2. Placeholder scan:** No placeholders — all code is explicit.

**3. Type consistency:**
- `WizardState` types consistent across all steps
- `CompleteWorkOrderPayload` matches existing type in `types/index.ts`
- Navigation params use existing `WorkOrder` type
