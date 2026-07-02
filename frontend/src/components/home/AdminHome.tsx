// ═══════════════════════════════════════════════════════════════════════
// AdminHome — Role-based home page for admin role
// UX-1.5: Role-Based Home Pages
//   - System Health (overall system status)
//   - Compliance Status (compliance checks summary)
//   - Audit Alerts (recent audit log entries)
//   - Skeleton loader while loading
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useDevices, useAlarms } from '../../hooks/useApiQuery';
import {
  Server,
  Shield,
  AlertTriangle,
  CheckCircle,
  Activity,
  HardDrive,
  Users as UsersIcon,
  ArrowRight,
  Clock,
  FileText,
} from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge, StatsCard } from '../../components/ui';
import type { Device } from '../../services/api/devices';
import type { Alarm } from '../../services/api/alarms';

// ─── Skeleton ────────────────────────────────────────────────────────

function AdminHomeSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-24 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="h-64 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        <div className="h-64 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      </div>
    </div>
  );
}

// ─── Main Component ──────────────────────────────────────────────────

export function AdminHome() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: devices, isLoading: devicesLoading } = useDevices();
  const { data: alarms, isLoading: alarmsLoading } = useAlarms();

  const deviceList = useMemo(() => {
    if (!devices) return [];
    return Array.isArray(devices) ? (devices as Device[]) : [];
  }, [devices]);

  const alarmList = useMemo(() => {
    if (!alarms) return [];
    return Array.isArray(alarms) ? (alarms as Alarm[]) : [];
  }, [alarms]);

  const onlineCount = useMemo(
    () => deviceList.filter((d) => d.status === 'online' || d.status === 'ONLINE').length,
    [deviceList],
  );

  const offlineCount = useMemo(
    () => deviceList.filter((d) => d.status === 'offline' || d.status === 'OFFLINE').length,
    [deviceList],
  );

  const warningCount = useMemo(
    () => deviceList.filter((d) => d.status === 'warning' || d.status === 'WARNING').length,
    [deviceList],
  );

  const criticalAlarms = useMemo(
    () => alarmList.filter((a) => a.priority >= 5).length,
    [alarmList],
  );

  const activeAlarms = useMemo(
    () => alarmList.length,
    [alarmList],
  );

  const isLoading = devicesLoading || alarmsLoading;

  // Compliance status (mock — replace with real API)
  const complianceItems = useMemo(
    () => [
      { label: t('data_encryption') || 'Data Encryption', status: 'pass' as const },
      { label: t('audit_logging') || 'Audit Logging', status: 'pass' as const },
      { label: t('access_control') || 'Access Control', status: 'pass' as const },
      { label: t('backup_status') || 'Backup Status', status: 'warn' as const },
      { label: t('certificate_expiry') || 'Certificate Expiry', status: 'pass' as const },
    ],
    [t],
  );

  if (isLoading) {
    return <AdminHomeSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* System Health Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('total_devices') || 'Total Devices'}
          value={deviceList.length}
          icon={HardDrive}
          iconBgColor="bg-indigo-50"
          iconColor="text-indigo-600"
        />
        <StatsCard
          title={t('online') || 'Online'}
          value={onlineCount}
          icon={Activity}
          iconBgColor="bg-emerald-50"
          iconColor="text-emerald-600"
        />
        <StatsCard
          title={t('offline') || 'Offline'}
          value={offlineCount}
          icon={Server}
          iconBgColor={offlineCount > 0 ? 'bg-red-50' : 'bg-slate-50'}
          iconColor={offlineCount > 0 ? 'text-red-600' : 'text-slate-400'}
        />
        <StatsCard
          title={t('critical_alarms') || 'Critical Alarms'}
          value={criticalAlarms}
          icon={AlertTriangle}
          iconBgColor={criticalAlarms > 0 ? 'bg-red-50' : 'bg-emerald-50'}
          iconColor={criticalAlarms > 0 ? 'text-red-600' : 'text-emerald-600'}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* System Health Map */}
        <Card>
          <CardHeader
            action={
              <Button
                variant="ghost"
                size="sm"
                icon={<ArrowRight className="w-4 h-4" />}
                onClick={() => navigate('/devices')}
              >
                {t('view_all') || 'View All'}
              </Button>
            }
          >
            <span className="flex items-center gap-2">
              <Activity className="w-4 h-4 text-blue-500" />
              {t('system_health') || 'System Health'}
            </span>
          </CardHeader>
          <CardBody>
            {/* Health Summary */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {t('devices_online') || 'Devices Online'}
                </span>
                <span className="text-sm font-semibold text-emerald-600 dark:text-emerald-400">
                  {onlineCount}/{deviceList.length}
                </span>
              </div>
              <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2.5">
                <div
                  className="bg-emerald-500 h-2.5 rounded-full transition-all"
                  style={{
                    width: `${deviceList.length > 0 ? (onlineCount / deviceList.length) * 100 : 0}%`,
                  }}
                />
              </div>

              <div className="flex items-center justify-between pt-2">
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {t('devices_warning') || 'Devices Warning'}
                </span>
                <span
                  className={`text-sm font-semibold ${warningCount > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-slate-500'}`}
                >
                  {warningCount}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {t('devices_offline') || 'Devices Offline'}
                </span>
                <span
                  className={`text-sm font-semibold ${offlineCount > 0 ? 'text-red-600 dark:text-red-400' : 'text-slate-500'}`}
                >
                  {offlineCount}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {t('active_alarms') || 'Active Alarms'}
                </span>
                <span
                  className={`text-sm font-semibold ${activeAlarms > 0 ? 'text-red-600 dark:text-red-400' : 'text-slate-500'}`}
                >
                  {activeAlarms}
                </span>
              </div>
            </div>
          </CardBody>
        </Card>

        {/* Compliance Status */}
        <Card>
          <CardHeader
            action={
              <Button
                variant="ghost"
                size="sm"
                icon={<ArrowRight className="w-4 h-4" />}
                onClick={() => navigate('/compliance-shield')}
              >
                {t('details') || 'Details'}
              </Button>
            }
          >
            <span className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-purple-500" />
              {t('compliance_status') || 'Compliance Status'}
            </span>
          </CardHeader>
          <CardBody>
            <div className="space-y-2">
              {complianceItems.map((item) => (
                <div
                  key={item.label}
                  className="flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-800 last:border-0"
                >
                  <span className="text-sm text-slate-700 dark:text-slate-300">
                    {item.label}
                  </span>
                  <Badge
                    variant={item.status === 'pass' ? 'success' : 'warning'}
                    size="sm"
                  >
                    {item.status === 'pass'
                      ? (t('compliant') || 'Compliant')
                      : (t('attention_needed') || 'Attention')}
                  </Badge>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>
      </div>

      {/* Audit Alerts */}
      <Card>
        <CardHeader
          action={
            <Button
              variant="ghost"
              size="sm"
              icon={<ArrowRight className="w-4 h-4" />}
              onClick={() => navigate('/audit-log')}
            >
              {t('view_all') || 'View All'}
            </Button>
          }
        >
          <span className="flex items-center gap-2">
            <FileText className="w-4 h-4 text-slate-500" />
            {t('recent_audit_alerts') || 'Recent Audit Alerts'}
          </span>
        </CardHeader>
        <CardBody>
          {alarmList.length === 0 ? (
            <div className="text-center py-8">
              <CheckCircle className="w-10 h-10 text-emerald-300 dark:text-emerald-600 mx-auto mb-3" />
              <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                {t('no_alarms') || 'No active alarms'}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                {t('system_operating_normally') || 'System is operating normally'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {alarmList.slice(0, 8).map((alarm) => (
                <div
                  key={`${alarm.device_id}-${alarm.timestamp}`}
                  className="flex items-center justify-between p-2 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800/70 transition-colors"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div
                      className={`w-2 h-2 rounded-full shrink-0 ${
                        alarm.priority >= 5
                          ? 'bg-red-500'
                          : alarm.priority >= 3
                            ? 'bg-amber-500'
                            : 'bg-blue-500'
                      }`}
                    />
                    <div className="min-w-0">
                      <p className="text-sm text-slate-800 dark:text-slate-200 truncate">
                        {alarm.description || alarm.device_id}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {alarm.device_id.slice(0, 12)}...
                        {alarm.timestamp
                          ? ` • ${new Date(alarm.timestamp).toLocaleString()}`
                          : ''}
                      </p>
                    </div>
                  </div>
                  <Badge
                    variant={
                      alarm.priority >= 5
                        ? 'danger'
                        : alarm.priority >= 3
                          ? 'warning'
                          : 'info'
                    }
                    size="sm"
                  >
                    P{alarm.priority}
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}

export default AdminHome;
