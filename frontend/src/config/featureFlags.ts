// ═══════════════════════════════════════════════════════════════════════
// featureFlags.ts — Централизованный конфиг feature flags
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — флаги не отключают security controls)
//   - OWASP ASVS V1.8 (Feature flags не раскрывают sensitive functionality)
// ═══════════════════════════════════════════════════════════════════════

export interface FeatureFlags {
  /** UX-1.2: Unified Work Hub с табами My Tasks / Team / Requests */
  unified_work_hub_v2: boolean;

  /** UX-3.2: Auto-fill TO Journals при закрытии WorkOrder */
  to_auto_generation: boolean;

  /** UX-4.2: QR Mobile Lifecycle — Onboarding / Maintenance / Verification */
  mobile_qr_lifecycle: boolean;

  /** UX-1.1: Sidebar Progressive Disclosure — 5 domain groups + 14 visible items */
  sidebar_progressive_disclosure: boolean;

  /** UX-2.1: Three-Column Detail Layout для Device Detail */
  three_column_detail_layout: boolean;

  /** UX-5.1: Command Palette with regulatory awareness */
  command_palette_regulatory: boolean;

  /** UX-3.4: AI Copilot for TO Journals */
  ai_copilot_to_journals: boolean;

  /** UX-3.5: Print Template Visual Editor */
  print_template_builder: boolean;

  /** UX-3.6: Hash-Chain Digital Signatures (СТБ 34.101.27) */
  hash_chain_signatures: boolean;

  /** UX-3.7: Regulatory Checklist Enforcement */
  regulatory_gatekeeper: boolean;

  /** UX-1.5: Role-Based Home Pages (Technician/Manager/Admin) */
  role_based_home_pages: boolean;

  /** UX-4.1: Asset Tree Drill-down with URL sync */
  asset_tree_drilldown: boolean;
}

const featureFlags: FeatureFlags = {
  unified_work_hub_v2: false,
  to_auto_generation: false,
  mobile_qr_lifecycle: false,
  sidebar_progressive_disclosure: false,
  three_column_detail_layout: false,
  command_palette_regulatory: false,
  ai_copilot_to_journals: false,
  print_template_builder: false,
  hash_chain_signatures: false,
  regulatory_gatekeeper: false,
  role_based_home_pages: true,
  asset_tree_drilldown: true,
};

/**
 * Получить значение feature flag.
 * В будущем — замена на удалённый конфиг (LaunchDarkly / ConfigCat / PostHog).
 */
export function getFeatureFlag<K extends keyof FeatureFlags>(flag: K): FeatureFlags[K] {
  return featureFlags[flag];
}

/**
 * Проверить включён ли флаг (type-safe).
 */
export function isFeatureEnabled(flag: keyof FeatureFlags): boolean {
  return featureFlags[flag] === true;
}

export default featureFlags;
