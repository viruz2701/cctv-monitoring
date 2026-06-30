// ═══════════════════════════════════════════════════════════════════════
// Store barrel export
// ARCH-02: Zustand stores for UI state + React Query for server state
// ═══════════════════════════════════════════════════════════════════════

export { useThemeStore, type Theme } from './themeStore';
export { useAlertStore, useToastAlerts, useAlertFilters, useSelectedAlertIds } from './alertStore';
export type { ToastAlert } from './alertStore';
export { useCommandPaletteStore } from './commandPaletteStore';
export { useOnboardingStore } from './onboardingStore';
export {
  useWorkspaceStore,
  type Workspace,
  type WidgetLayout,
} from './workspaceStore';
export {
  useFilterStore,
  type SavedView as FilterSavedView,
} from './filterStore';
export {
  useSavedViewsStore,
  type SavedView as DashboardSavedView,
} from './savedViewsStore';
export {
  useReportsStore,
  startReportExpirationSweep,
  stopReportExpirationSweep,
} from './reportsStore';
export type { ReportHistoryItem } from './reportsStore';
export { useSettingsStore } from './settingsStore';

// ── ARCH.1: Theme Provider (migrated from context/ThemeContext) ─────
export { ThemeProvider, useTheme } from './ThemeProvider';

// ── ARCH.1: New stores ──────────────────────────────────────────────
export { useAuthStore, type AuthUser } from './authStore';
export type { AuthState } from './authStore';
export { useUIStore, type UIState, type ModalConfig, type PanelState } from './uiStore';
export {
  useNotificationStore,
  type NotificationItem,
  type NotificationType,
  type NotificationFilter,
} from './notificationStore';

// ── PROTO-06: Descriptor Editor State ──────────────────────────────
export {
  useDescriptorStore,
  useDescriptorMode,
  useDescriptorTab,
  useDescriptorList,
  useCurrentDescriptor,
  useDescriptorLoading,
  useDescriptorError,
  useDescriptorDirty,
  createEmptyDescriptor,
} from './descriptorStore';
export type { EditorMode, DescriptorViewTab } from '../types/descriptor';

// ── PROTO-07: Community Protocol Registry ─────────────────────────
export {
  useCommunityRegistryStore,
  useCommunityDescriptors,
  useCommunityDescriptorDetail,
  useCommunityRegistryLoading,
  useCommunityRegistryError,
  useCommunityRegistryPagination,
  useCommunityRegistryFilter,
} from './communityRegistryStore';
