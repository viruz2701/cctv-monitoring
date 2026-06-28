// ═══════════════════════════════════════════════════════════════════════════
// SecurityAdvisories — Vulnerability Disclosure Program (VDP) Page
//
// P0-N2: Vulnerability Disclosure Program
// Соответствует: EU CRA, ISO 27001 A.6.1, OWASP ASVS V1.4
//
// Отображает:
//   - Текущие CVE advisories
//   - Историю раскрытия уязвимостей
//   - RSS фид для подписки
//   - Hall of Fame
// ═══════════════════════════════════════════════════════════════════════════
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Shield,
  ShieldAlert,
  Clock,
  ExternalLink,
  Rss,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Bug,
  Users,
} from 'lucide-react';

// ── Types ────────────────────────────────────────────────────────────────

interface Advisory {
  id: string;
  cveId: string;
  title: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: 'fixed' | 'in_progress' | 'acknowledged';
  affectedVersions: string;
  fixedVersion: string;
  reportedBy: string;
  reportedAt: string;
  fixedAt: string | null;
  description: string;
  impact: string;
  workaround: string | null;
  cvss: number;
}

interface HallOfFameEntry {
  name: string;
  contribution: string;
  date: string;
}

// ── Mock Data ────────────────────────────────────────────────────────────

const MOCK_ADVISORIES: Advisory[] = [
  {
    id: 'GHSA-001',
    cveId: 'CVE-2026-0001',
    title: 'SQL Injection in Device Search Endpoint',
    severity: 'critical',
    status: 'fixed',
    affectedVersions: '< 1.2.0',
    fixedVersion: '1.2.0',
    reportedBy: 'security-researcher@example.com',
    reportedAt: '2026-03-15',
    fixedAt: '2026-03-22',
    description:
      'The device search endpoint /api/v1/devices/search was vulnerable to SQL injection via unsanitized query parameters. An authenticated attacker could execute arbitrary SQL.',
    impact:
      'An authenticated attacker with low privileges could extract sensitive data from the database, including user credentials and device secrets.',
    workaround:
      'Upgrade to v1.2.0. If immediate upgrade is not possible, restrict access to the device search endpoint to admin users only.',
    cvss: 9.1,
  },
  {
    id: 'GHSA-002',
    cveId: 'CVE-2026-0002',
    title: 'Stored XSS in Work Order Notes',
    severity: 'high',
    status: 'fixed',
    affectedVersions: '< 1.1.5',
    fixedVersion: '1.1.5',
    reportedBy: 'pentest-team@gb-telemetry.com',
    reportedAt: '2026-02-01',
    fixedAt: '2026-02-10',
    description:
      'Work order notes were not sanitized before rendering, allowing stored XSS attacks. An attacker with technician privileges could inject malicious scripts.',
    impact:
      'An attacker could execute arbitrary JavaScript in the context of other users viewing the affected work order.',
    workaround:
      'Disable rich text editing for work order notes in admin settings.',
    cvss: 7.5,
  },
  {
    id: 'GHSA-003',
    cveId: 'CVE-2026-0003',
    title: 'JWT Token Validation Bypass',
    severity: 'critical',
    status: 'in_progress',
    affectedVersions: '< 1.3.0',
    fixedVersion: '1.3.0',
    reportedBy: 'external-researcher',
    reportedAt: '2026-06-20',
    fixedAt: null,
    description:
      'A vulnerability in JWT token validation allows attackers to forge tokens with modified claims when using the legacy HS256 algorithm.',
    impact:
      'An attacker could forge authentication tokens and impersonate any user, including administrators.',
    workaround:
      'Ensure only RS256 or ES256 algorithms are configured for JWT signing. Disable HS256 in server configuration.',
    cvss: 9.8,
  },
];

const MOCK_HALL_OF_FAME: HallOfFameEntry[] = [
  {
    name: 'Alex S.',
    contribution: 'SQL Injection — Device Search (CVE-2026-0001)',
    date: '2026-03',
  },
  {
    name: 'Maria K.',
    contribution: 'Stored XSS — Work Order Notes (CVE-2026-0002)',
    date: '2026-02',
  },
];

// ── Severity Badge ───────────────────────────────────────────────────────

const SeverityBadge: React.FC<{ severity: Advisory['severity'] }> = ({
  severity,
}) => {
  const colors: Record<Advisory['severity'], string> = {
    critical:
      'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border-red-200 dark:border-red-800',
    high: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400 border-orange-200 dark:border-orange-800',
    medium:
      'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800',
    low: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-green-200 dark:border-green-800',
  };

  return (
    <span
      className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium border ${colors[severity]}`}
    >
      {severity === 'critical' && <ShieldAlert className="w-3 h-3" />}
      {severity === 'high' && <AlertTriangle className="w-3 h-3" />}
      {severity === 'medium' && <Shield className="w-3 h-3" />}
      {severity === 'low' && <CheckCircle2 className="w-3 h-3" />}
      {severity.toUpperCase()}
    </span>
  );
};

// ── Status Badge ─────────────────────────────────────────────────────────

const StatusBadge: React.FC<{ status: Advisory['status'] }> = ({ status }) => {
  const config: Record<
    Advisory['status'],
    { label: string; color: string; icon: React.ReactNode }
  > = {
    fixed: {
      label: 'Fixed',
      color:
        'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800',
      icon: <CheckCircle2 className="w-3 h-3" />,
    },
    in_progress: {
      label: 'In Progress',
      color:
        'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-blue-200 dark:border-blue-800',
      icon: <Clock className="w-3 h-3" />,
    },
    acknowledged: {
      label: 'Acknowledged',
      color:
        'bg-slate-100 text-slate-800 dark:bg-slate-900/30 dark:text-slate-400 border-slate-200 dark:border-slate-800',
      icon: <Shield className="w-3 h-3" />,
    },
  };

  const c = config[status];
  return (
    <span
      className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium border ${c.color}`}
    >
      {c.icon}
      {c.label}
    </span>
  );
};

// ── Advisory Card ────────────────────────────────────────────────────────

const AdvisoryCard: React.FC<{ advisory: Advisory }> = ({ advisory }) => {
  const [expanded, setExpanded] = useState(false);

  const cvssColor =
    advisory.cvss >= 9
      ? 'text-red-600 dark:text-red-400'
      : advisory.cvss >= 7
        ? 'text-orange-600 dark:text-orange-400'
        : advisory.cvss >= 4
          ? 'text-yellow-600 dark:text-yellow-400'
          : 'text-green-600 dark:text-green-400';

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full text-left px-5 py-4 flex items-start gap-3 hover:bg-slate-50 dark:hover:bg-slate-750 transition-colors"
        aria-expanded={expanded}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
              {advisory.cveId}
            </span>
            <SeverityBadge severity={advisory.severity} />
            <StatusBadge status={advisory.status} />
            <span className={`text-sm font-semibold ${cvssColor}`}>
              CVSS {advisory.cvss.toFixed(1)}
            </span>
          </div>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
            {advisory.title}
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            Reported: {advisory.reportedAt}
            {advisory.fixedAt && ` · Fixed: ${advisory.fixedAt}`}
            {advisory.status === 'in_progress' && ' · Patch in progress'}
          </p>
        </div>
        <div className="flex-shrink-0 mt-1">
          {expanded ? (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronRight className="w-4 h-4 text-slate-400" />
          )}
        </div>
      </button>

      {expanded && (
        <div className="px-5 pb-4 border-t border-slate-100 dark:border-slate-700">
          <div className="mt-3 space-y-3">
            <Section label="Description" text={advisory.description} />
            <Section label="Impact" text={advisory.impact} />
            {advisory.workaround && (
              <Section label="Workaround" text={advisory.workaround} />
            )}
            <div className="grid grid-cols-2 gap-3 text-xs">
              <div>
                <span className="text-slate-500 dark:text-slate-400">
                  Affected versions:{' '}
                </span>
                <span className="font-mono text-slate-700 dark:text-slate-300">
                  {advisory.affectedVersions}
                </span>
              </div>
              <div>
                <span className="text-slate-500 dark:text-slate-400">
                  Fixed in:{' '}
                </span>
                <span className="font-mono text-slate-700 dark:text-slate-300">
                  {advisory.fixedVersion}
                </span>
              </div>
              <div>
                <span className="text-slate-500 dark:text-slate-400">
                  Reported by:{' '}
                </span>
                <span className="text-slate-700 dark:text-slate-300">
                  {advisory.reportedBy}
                </span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ── Section Helper ───────────────────────────────────────────────────────

const Section: React.FC<{ label: string; text: string }> = ({
  label,
  text,
}) => (
  <div>
    <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-1">
      {label}
    </h4>
    <p className="text-sm text-slate-700 dark:text-slate-300 leading-relaxed">
      {text}
    </p>
  </div>
);

// ── Main Component ───────────────────────────────────────────────────────

export const SecurityAdvisories: React.FC = () => {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<Advisory['severity'] | 'all'>('all');
  const [statusFilter, setStatusFilter] = useState<Advisory['status'] | 'all'>(
    'all',
  );

  const filteredAdvisories = MOCK_ADVISORIES.filter((a) => {
    if (filter !== 'all' && a.severity !== filter) return false;
    if (statusFilter !== 'all' && a.status !== statusFilter) return false;
    return true;
  });

  const openCount = MOCK_ADVISORIES.filter(
    (a) => a.status !== 'fixed',
  ).length;

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <Shield className="w-8 h-8 text-emerald-600 dark:text-emerald-400" />
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {t('security_advisories') || 'Security Advisories'}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {t('security_advisories_desc') ||
                'Vulnerability Disclosure Program — Coordinated Disclosure'}
            </p>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mt-6">
          <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {MOCK_ADVISORIES.length}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('total_advisories') || 'Total Advisories'}
            </div>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
            <div className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">
              {MOCK_ADVISORIES.filter((a) => a.status === 'fixed').length}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('fixed') || 'Fixed'}
            </div>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
            <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">
              {openCount}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('open') || 'Open'}
            </div>
          </div>
        </div>

        {/* RSS Feed Link */}
        <div className="flex items-center gap-2 mt-4 text-xs">
          <Rss className="w-4 h-4 text-orange-500" />
          <a
            href="/security-advisories/feed.xml"
            className="text-emerald-600 dark:text-emerald-400 hover:underline font-medium"
          >
            {t('rss_feed') || 'RSS Feed'}
          </a>
          <span className="text-slate-400">·</span>
          <a
            href="/.well-known/security.txt"
            className="text-emerald-600 dark:text-emerald-400 hover:underline font-medium"
          >
            security.txt
          </a>
          <span className="text-slate-400">·</span>
          <a
            href="/SECURITY.md"
            className="text-emerald-600 dark:text-emerald-400 hover:underline font-medium"
          >
            {t('full_policy') || 'Full Policy'}
          </a>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2 mb-4 flex-wrap">
        <span className="text-xs font-medium text-slate-500 dark:text-slate-400 mr-1">
          {t('severity') || 'Severity'}:
        </span>
        {(['all', 'critical', 'high', 'medium', 'low'] as const).map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-colors ${
              filter === s
                ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400 border border-transparent hover:bg-slate-200 dark:hover:bg-slate-700'
            }`}
          >
            {s === 'all' ? 'All' : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}

        <span className="text-xs font-medium text-slate-500 dark:text-slate-400 ml-3 mr-1">
          {t('status') || 'Status'}:
        </span>
        {(['all', 'fixed', 'in_progress', 'acknowledged'] as const).map(
          (s) => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-colors ${
                statusFilter === s
                  ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800'
                  : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400 border border-transparent hover:bg-slate-200 dark:hover:bg-slate-700'
              }`}
            >
              {s === 'all'
                ? 'All'
                : s === 'in_progress'
                  ? 'In Progress'
                  : s.charAt(0).toUpperCase() + s.slice(1)}
            </button>
          ),
        )}
      </div>

      {/* Advisories List */}
      <div className="space-y-3">
        {filteredAdvisories.length === 0 ? (
          <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-8 text-center">
            <Shield className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {t('no_advisories') ||
                'No advisories match the selected filters.'}
            </p>
          </div>
        ) : (
          filteredAdvisories.map((advisory) => (
            <AdvisoryCard key={advisory.id} advisory={advisory} />
          ))
        )}
      </div>

      {/* Hall of Fame */}
      <div className="mt-10">
        <div className="flex items-center gap-2 mb-4">
          <Users className="w-5 h-5 text-amber-500" />
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            {t('hall_of_fame') || 'Hall of Fame'}
          </h2>
        </div>

        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
          {MOCK_HALL_OF_FAME.length === 0 ? (
            <div className="p-6 text-center">
              <Bug className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
              <p className="text-sm text-slate-500 dark:text-slate-400">
                {t('hall_of_fame_empty') ||
                  'No entries yet — be the first to report a vulnerability!'}
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100 dark:divide-slate-700">
              {MOCK_HALL_OF_FAME.map((entry, i) => (
                <div
                  key={i}
                  className="px-5 py-3 flex items-center justify-between"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center">
                      <span className="text-xs font-bold text-amber-700 dark:text-amber-400">
                        #{i + 1}
                      </span>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        {entry.name}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {entry.contribution}
                      </p>
                    </div>
                  </div>
                  <span className="text-xs text-slate-400">{entry.date}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Report CTA */}
      <div className="mt-10 bg-emerald-50 dark:bg-emerald-900/20 rounded-xl border border-emerald-200 dark:border-emerald-800 p-6 text-center">
        <Bug className="w-8 h-8 text-emerald-600 dark:text-emerald-400 mx-auto mb-3" />
        <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-2">
          {t('report_vulnerability') || 'Found a Vulnerability?'}
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-4 max-w-lg mx-auto">
          {t('report_vulnerability_desc') ||
            'We encourage responsible disclosure. Report vulnerabilities to our security team and help make CCTV Health Monitor safer for everyone.'}
        </p>
        <a
          href="mailto:security@gb-telemetry.com"
          className="inline-flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <ExternalLink className="w-4 h-4" />
          {t('report_now') || 'Report Now'}
        </a>
      </div>
    </div>
  );
};

export default SecurityAdvisories;
