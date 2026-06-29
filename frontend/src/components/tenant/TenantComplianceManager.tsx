// ═══════════════════════════════════════════════════════════════════════
// TenantComplianceManager — Admin UI for per-tenant compliance region (P0-CE.5)
//
// Позволяет администратору:
//   - Просматривать compliance регион tenant'ов
//   - Устанавливать compliance регион при создании tenant
//   - Блокировать регион (immutable после first data)
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - ISO 27001 A.8.1 (Asset management — tenant classification)
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//   - OWASP ASVS V4 (RBAC — admin only)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect, useCallback } from 'react';
import { request } from '../../services/api';
import { Shield, Lock, Unlock, AlertTriangle, CheckCircle } from '../ui/Icons';
import { useTranslation } from 'react-i18next';

// ── Types ────────────────────────────────────────────────────────────

export interface ComplianceRegion {
  region: string;
  name: string;
  description?: string;
}

export interface TenantCompliance {
  tenant_id: string;
  compliance_region: string;
  compliance_locked: boolean;
}

interface TenantComplianceManagerProps {
  tenantId?: string; // если указан — режим single-tenant, иначе список
}

// ── Constants ────────────────────────────────────────────────────────

const REGION_LABELS: Record<string, { name: string; description: string }> = {
  BY: {
    name: 'Belarus (СТБ)',
    description: 'СТБ 34.101.27, СТБ 34.101.30, Приказ ОАЦ №66',
  },
  EU: {
    name: 'European Union (GDPR)',
    description: 'GDPR, NIS2, eIDAS',
  },
  INTL: {
    name: 'International',
    description: 'ISO 27001, ISO 27019, IEC 62443',
  },
  RU: {
    name: 'Russia (ГОСТ)',
    description: 'ГОСТ, 152-ФЗ, ФСТЭК',
  },
  CN: {
    name: 'China (SM)',
    description: 'SM2/SM3/SM4, MLPS 2.0',
  },
  US: {
    name: 'United States (FIPS)',
    description: 'FIPS 140-3, HIPAA, SOC 2',
  },
};

// ── Component ────────────────────────────────────────────────────────

export function TenantComplianceManager({ tenantId }: TenantComplianceManagerProps) {
  const { t } = useTranslation();
  const [compliance, setCompliance] = useState<TenantCompliance | null>(null);
  const [regions, setRegions] = useState<ComplianceRegion[]>([]);
  const [selectedRegion, setSelectedRegion] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // ── Load regions list ─────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;

    async function loadRegions() {
      try {
        const data = await request<ComplianceRegion[]>('/admin/tenants/compliance/regions');
        if (!cancelled) {
          setRegions(data);
        }
      } catch {
        // Fallback to static list
        const fallback: ComplianceRegion[] = Object.entries(REGION_LABELS).map(([region, info]) => ({
          region,
          name: info.name,
          description: info.description,
        }));
        if (!cancelled) {
          setRegions(fallback);
        }
      }
    }

    loadRegions();
    return () => { cancelled = true; };
  }, []);

  // ── Load tenant compliance (if tenantId provided) ─────────────────
  useEffect(() => {
    if (!tenantId) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function loadCompliance() {
      try {
        const data = await request<TenantCompliance>(`/admin/tenants/${tenantId}/compliance`);
        if (!cancelled) {
          setCompliance(data);
          setSelectedRegion(data.compliance_region);
        }
      } catch (err) {
        if (!cancelled) {
          setError(`Failed to load compliance: ${err instanceof Error ? err.message : 'Unknown error'}`);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadCompliance();
    return () => { cancelled = true; };
  }, [tenantId]);

  // ── Set compliance region ─────────────────────────────────────────
  const handleSetRegion = useCallback(async () => {
    if (!tenantId || !selectedRegion) return;

    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      const data = await request<TenantCompliance>(`/admin/tenants/${tenantId}/compliance`, {
        method: 'PUT',
        body: JSON.stringify({ region: selectedRegion }),
      });
      setCompliance(data);
      setSuccess(t('compliance_region_updated') || 'Compliance region updated successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update compliance region');
    } finally {
      setSaving(false);
    }
  }, [tenantId, selectedRegion, t]);

  // ── Lock compliance region ────────────────────────────────────────
  const handleLock = useCallback(async () => {
    if (!tenantId) return;

    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      const data = await request<TenantCompliance>(`/admin/tenants/${tenantId}/compliance/lock`, {
        method: 'POST',
      });
      setCompliance(data);
      setSuccess(t('compliance_locked') || 'Compliance region locked');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to lock compliance region');
    } finally {
      setSaving(false);
    }
  }, [tenantId, t]);

  if (!tenantId) {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5 text-blue-500" />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {t('tenant_compliance') || 'Tenant Compliance'}
          </h2>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t('tenant_compliance_description') || 'Select a tenant to manage its compliance region.'}
        </p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
        <div className="animate-pulse space-y-3">
          <div className="h-5 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
          <div className="h-10 bg-gray-200 dark:bg-gray-700 rounded w-full" />
        </div>
      </div>
    );
  }

  // ── Render ────────────────────────────────────────────────────────
  const isLocked = compliance?.compliance_locked ?? false;
  const currentRegion = compliance?.compliance_region || '';

  return (
    <div className="p-6 bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="flex items-center gap-2 mb-6">
        <Shield className="w-5 h-5 text-blue-500" />
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          {t('tenant_compliance') || 'Tenant Compliance'}
        </h2>
        {isLocked ? (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
            <Lock className="w-3 h-3" />
            {t('locked') || 'Locked'}
          </span>
        ) : (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400">
            <Unlock className="w-3 h-3" />
            {t('editable') || 'Editable'}
          </span>
        )}
      </div>

      {/* Tenant ID */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {t('tenant_id') || 'Tenant ID'}
        </label>
        <code className="text-sm bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-gray-900 dark:text-gray-100">
          {tenantId}
        </code>
      </div>

      {/* Current region display */}
      {currentRegion && (
        <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('current_compliance_region') || 'Current Compliance Region'}
          </label>
          <div className="flex items-center gap-2">
            <span className="text-lg font-bold text-gray-900 dark:text-gray-100">
              {currentRegion}
            </span>
            <span className="text-sm text-gray-500 dark:text-gray-400">
              — {REGION_LABELS[currentRegion]?.name || currentRegion}
            </span>
          </div>
          {REGION_LABELS[currentRegion]?.description && (
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {REGION_LABELS[currentRegion].description}
            </p>
          )}
        </div>
      )}

      {/* Region selector (disabled if locked) */}
      {!isLocked && (
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            {t('select_compliance_region') || 'Select Compliance Region'}
          </label>
          <div className="space-y-2">
            {regions.map((r) => (
              <label
                key={r.region}
                className={`
                  flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors
                  ${selectedRegion === r.region
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                  }
                `}
              >
                <input
                  type="radio"
                  name="compliance_region"
                  value={r.region}
                  checked={selectedRegion === r.region}
                  onChange={() => setSelectedRegion(r.region)}
                  className="sr-only"
                />
                <div className={`
                  w-4 h-4 rounded-full border-2 flex items-center justify-center flex-shrink-0
                  ${selectedRegion === r.region
                    ? 'border-blue-500'
                    : 'border-gray-300 dark:border-gray-600'
                  }
                `}>
                  {selectedRegion === r.region && (
                    <div className="w-2 h-2 rounded-full bg-blue-500" />
                  )}
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {r.name}
                  </span>
                  {r.description && (
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                      {r.description}
                    </p>
                  )}
                </div>
              </label>
            ))}
          </div>

          {/* Save button */}
          <button
            onClick={handleSetRegion}
            disabled={saving || !selectedRegion || selectedRegion === currentRegion}
            className="mt-4 w-full px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium
              hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {saving
              ? (t('saving') || 'Saving...')
              : (t('save_compliance_region') || 'Save Compliance Region')}
          </button>

          {/* Warning about immutability */}
          <div className="mt-3 flex items-start gap-2 text-xs text-amber-600 dark:text-amber-400">
            <AlertTriangle className="w-4 h-4 flex-shrink-0 mt-0.5" />
            <p>
              {t('compliance_immutable_warning') ||
                'The compliance region cannot be changed after the first data is created for this tenant.'}
            </p>
          </div>
        </div>
      )}

      {/* Lock button (only when not locked) */}
      {!isLocked && currentRegion && (
        <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={handleLock}
            disabled={saving}
            className="w-full px-4 py-2 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400
              rounded-lg text-sm font-medium hover:bg-red-100 dark:hover:bg-red-900/40
              disabled:opacity-50 disabled:cursor-not-allowed transition-colors
              border border-red-200 dark:border-red-800"
          >
            <Lock className="w-4 h-4 inline mr-1" />
            {t('lock_compliance_region') || 'Lock Compliance Region'}
          </button>
        </div>
      )}

      {/* Status messages */}
      {error && (
        <div className="mt-4 flex items-center gap-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <AlertTriangle className="w-4 h-4 text-red-500 flex-shrink-0" />
          <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
        </div>
      )}

      {success && (
        <div className="mt-4 flex items-center gap-2 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
          <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
          <p className="text-sm text-green-700 dark:text-green-400">{success}</p>
        </div>
      )}
    </div>
  );
}
