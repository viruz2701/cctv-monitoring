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

  /**
   * Добавляй новые флаги сюда.
   * Соглашение: snake_case с суффиксом версии (_v2, _v3).
   * Значение по умолчанию — false (feature gated).
   */
}

const featureFlags: FeatureFlags = {
  unified_work_hub_v2: false,
  to_auto_generation: false,
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
