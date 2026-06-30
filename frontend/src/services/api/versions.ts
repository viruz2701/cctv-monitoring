// ═══════════════════════════════════════════════════════════════════════
// API Versions — P2-API Versioning Strategy
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface VersionInfo {
  version: string;
  deprecated: boolean;
  sunset?: string; // RFC 3339
  released_at: string;
  deprecated_at?: string;
  changelog: string;
}

export interface ChangelogEntry {
  version: string;
  date: string;
  change: string;
  deprecated: boolean;
  sunset?: string;
}

export interface VersionListResponse {
  versions: VersionInfo[];
  total: number;
}

export interface ChangelogResponse {
  changelog: ChangelogEntry[];
  total: number;
}

// ─── API Methods ────────────────────────────────────────────────────

export const versionsApi = {
  /** GET /api/v1/versions — список всех версий API */
  listVersions(): Promise<VersionListResponse> {
    return request<VersionListResponse>('/versions');
  },

  /** POST /api/v1/versions — создать новую версию (admin) */
  createVersion(version: string, changelog: string): Promise<{ version: string; status: string }> {
    return request<{ version: string; status: string }>('/versions', {
      method: 'POST',
      body: JSON.stringify({ version, changelog }),
    });
  },

  /** PUT /api/v1/versions/{version} — обновить метаданные версии (admin) */
  updateVersion(
    version: string,
    data: {
      deprecated?: boolean;
      sunset?: string;
      deprecated_at?: string;
      changelog?: string;
    },
  ): Promise<{ version: string; status: string }> {
    return request<{ version: string; status: string }>(`/versions/${version}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  /** GET /api/v1/changelog — changelog API */
  getChangelog(): Promise<ChangelogResponse> {
    return request<ChangelogResponse>('/changelog');
  },
};
