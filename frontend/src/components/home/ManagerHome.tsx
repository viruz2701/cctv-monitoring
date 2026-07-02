// ═══════════════════════════════════════════════════════════════════════
// ManagerHome — Role-based home page for manager role
// UX-1.5: Role-Based Home Pages
//   - Team Heatmap (SLA summary for all technicians)
//   - SLA Breach risk (top at-risk work orders)
//   - Approvals pending (work orders needing approval)
//   - Skeleton loader while loading
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useWorkOrders } from '../../hooks/useApiQuery';
import {
  Users,
  AlertTriangle,
  CheckSquare,
  Activity,
  TrendingUp,
  Clock,
  ArrowRight,
  Shield,
  CheckCircle2,
  XCircle,
} from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge, StatsCard } from '../../components/ui';
import type { WorkOrder } from '../../services/workOrdersApi';

// ─── Skeleton ────────────────────────────────────────────────────────

function ManagerHomeSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-24 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        ))}
      </div>
      <div className="h-72 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        <div className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      </div>
    </div>
  );
}

// ─── Helpers ─────────────────────────────────────────────────────────

function isAtRisk(wo: WorkOrder): boolean {
  if (!wo.sla_deadline) return false;
  const deadline = new Date(wo.sla_deadline);
  const now = new Date();
  const hoursLeft = (deadline.getTime() - now.getTime()) / (1000 * 60 * 60);
  return hoursLeft > 0 && hoursLeft < 24;
}

function isBreached(wo: WorkOrder): boolean {
  if (!wo.sla_deadline) return false;
  return new Date(wo.sla_deadline) < new Date();
}

// ─── Main Component ──────────────────────────────────────────────────

export function ManagerHome() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: workOrders, isLoading } = useWorkOrders();

  const allOrders = useMemo(() => {
    if (!workOrders) return [];
    return Array.isArray(workOrders) ? workOrders : [];
  }, [workOrders]);

  const slaBreachRisk = useMemo(() => {
    return allOrders.filter(
      (wo) =>
        (wo.status === 'open' || wo.status === 'in_progress') &&
        (isAtRisk(wo) || isBreached(wo)),
    ).length;
  }, [allOrders]);

  const pendingApprovals = useMemo(() => {
    return allOrders.filter((wo) => wo.status === 'completed' && !wo.completed_at).length;
  }, [allOrders]);

  const activeOrders = useMemo(() => {
    return allOrders.filter(
      (wo) => wo.status === 'open' || wo.status === 'in_progress',
    ).length;
  }, [allOrders]);

  const atRiskOrders = useMemo(() => {
    return allOrders.filter(
      (wo) =>
        (wo.status === 'open' || wo.status === 'in_progress') &&
        (isAtRisk(wo) || isBreached(wo)),
    ).slice(0, 5);
  }, [allOrders]);

  if (isLoading) {
    return <ManagerHomeSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('active_orders') || 'Active Orders'}
          value={activeOrders}
          icon={Activity}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('sla_breach_risk') || 'SLA Breach Risk'}
          value={slaBreachRisk}
          icon={AlertTriangle}
          iconBgColor={slaBreachRisk > 0 ? 'bg-red-50' : 'bg-emerald-50'}
          iconColor={slaBreachRisk > 0 ? 'text-red-600' : 'text-emerald-600'}
        />
        <StatsCard
          title={t('pending_approvals') || 'Pending Approvals'}
          value={pendingApprovals}
          icon={CheckSquare}
          iconBgColor="bg-amber-50"
          iconColor="text-amber-600"
        />
        <StatsCard
          title={t('team_members') || 'Team SLA'}
          value={t('view_heatmap') || 'View Heatmap'}
          icon={Users}
          iconBgColor="bg-purple-50"
          iconColor="text-purple-600"
        />
      </div>

      {/* SLA Summary */}
      <Card>
        <CardHeader>
          {t('team_sla_summary') || 'Team SLA Summary'}
        </CardHeader>
        <CardBody>
          <div className="space-y-3">
            {[
              {
                label: t('on_track') || 'On Track',
                count: allOrders.filter((wo) => wo.sla_status === 'on_track' || !wo.sla_deadline).length,
                color: 'text-emerald-600',
                bg: 'bg-emerald-500',
                icon: CheckCircle2,
              },
              {
                label: t('at_risk') || 'At Risk',
                count: allOrders.filter(
                  (wo) =>
                    wo.sla_deadline &&
                    !isBreached(wo) &&
                    (wo.status === 'open' || wo.status === 'in_progress') &&
                    isAtRisk(wo),
                ).length,
                color: 'text-amber-600',
                bg: 'bg-amber-500',
                icon: AlertTriangle,
              },
              {
                label: t('breached') || 'Breached',
                count: allOrders.filter((wo) => wo.sla_deadline && isBreached(wo)).length,
                color: 'text-red-600',
                bg: 'bg-red-500',
                icon: XCircle,
              },
            ].map((item) => {
              const Icon = item.icon;
              const total = allOrders.filter((wo) => wo.sla_deadline).length || 1;
              const pct = Math.round((item.count / total) * 100);
              return (
                <div key={item.label} className="flex items-center gap-3">
                  <Icon className={`w-5 h-5 ${item.color} shrink-0`} />
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm text-slate-700 dark:text-slate-300">{item.label}</span>
                      <span className={`text-sm font-semibold ${item.color}`}>{item.count}</span>
                    </div>
                    <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-1.5">
                      <div
                        className={`${item.bg} h-1.5 rounded-full transition-all`}
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </CardBody>
      </Card>

      {/* Two-column: SLA Risk + Approvals */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* SLA Breach Risk */}
        <Card>
          <CardHeader
            action={
              <Button
                variant="ghost"
                size="sm"
                icon={<ArrowRight className="w-4 h-4" />}
                onClick={() => navigate('/sla')}
              >
                {t('view_all') || 'View All'}
              </Button>
            }
          >
            <span className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-red-500" />
              {t('sla_breach_risk') || 'SLA Breach Risk'}
            </span>
          </CardHeader>
          <CardBody>
            {atRiskOrders.length === 0 ? (
              <div className="text-center py-8">
                <Shield className="w-10 h-10 text-emerald-300 dark:text-emerald-600 mx-auto mb-3" />
                <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                  {t('no_sla_risks') || 'No SLA risks'}
                </p>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {t('all_orders_on_track') || 'All work orders are on track'}
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {atRiskOrders.map((wo) => (
                  <div
                    key={wo.id}
                    onClick={() => navigate(`/work-orders/${wo.id}`)}
                    className="flex items-center justify-between p-3 rounded-lg bg-red-50 dark:bg-red-900/20 hover:bg-red-100 dark:hover:bg-red-900/30 cursor-pointer transition-colors border border-red-200 dark:border-red-800/50"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <Clock className="w-4 h-4 text-red-500 shrink-0" />
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-red-800 dark:text-red-300 truncate">
                          {wo.device_name || wo.device_id || wo.id.slice(0, 8)}
                        </p>
                        <p className="text-xs text-red-600 dark:text-red-400">
                          {wo.sla_deadline
                            ? `${t('deadline') || 'Deadline'}: ${new Date(wo.sla_deadline).toLocaleDateString()}`
                            : t('no_deadline') || 'No deadline'}
                        </p>
                      </div>
                    </div>
                    <Badge
                      variant={isBreached(wo) ? 'danger' : 'warning'}
                      size="sm"
                    >
                      {isBreached(wo)
                        ? (t('breached') || 'Breached')
                        : (t('at_risk') || 'At Risk')}
                    </Badge>
                  </div>
                ))}
              </div>
            )}
          </CardBody>
        </Card>

        {/* Pending Approvals */}
        <Card>
          <CardHeader
            action={
              <Button
                variant="ghost"
                size="sm"
                icon={<ArrowRight className="w-4 h-4" />}
                onClick={() => navigate('/work-orders?status=completed')}
              >
                {t('view_all') || 'View All'}
              </Button>
            }
          >
            <span className="flex items-center gap-2">
              <CheckSquare className="w-4 h-4 text-amber-500" />
              {t('approvals_pending') || 'Approvals Pending'}
            </span>
          </CardHeader>
          <CardBody>
            {pendingApprovals === 0 ? (
              <div className="text-center py-8">
                <CheckSquare className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                  {t('no_pending_approvals') || 'No pending approvals'}
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {allOrders
                  .filter((wo) => wo.status === 'completed' && !wo.completed_at)
                  .slice(0, 5)
                  .map((wo) => (
                    <div
                      key={wo.id}
                      onClick={() => navigate(`/work-orders/${wo.id}`)}
                      className="flex items-center justify-between p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 hover:bg-amber-100 dark:hover:bg-amber-900/30 cursor-pointer transition-colors border border-amber-200 dark:border-amber-800/50"
                    >
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-amber-800 dark:text-amber-300 truncate">
                          {wo.device_name || wo.device_id || wo.id.slice(0, 8)}
                        </p>
                        <p className="text-xs text-amber-600 dark:text-amber-400">
                          {wo.assigned_to
                            ? `${t('by') || 'By'} ${wo.assignee_name || wo.assigned_to.slice(0, 8)}`
                            : t('unassigned') || 'Unassigned'}
                        </p>
                      </div>
                      <Badge variant="warning" size="sm">
                        {t('pending') || 'Pending'}
                      </Badge>
                    </div>
                  ))}
              </div>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  );
}

export default ManagerHome;
