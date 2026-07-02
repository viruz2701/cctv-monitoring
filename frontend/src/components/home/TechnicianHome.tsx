// ═══════════════════════════════════════════════════════════════════════
// TechnicianHome — Role-based home page for technician role
// UX-1.5: Role-Based Home Pages
//   - My Tasks (Today) — list of work orders assigned to current user
//   - Overdue — SLA-breached work orders
//   - Quick QR scan — button to open QR scanner
//   - Skeleton loader while loading
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { useWorkOrders } from '../../hooks/useApiQuery';
import {
  ClipboardList,
  AlertTriangle,
  QrCode,
  ArrowRight,
  Clock,
  CheckCircle2,
  Loader2,
} from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge, StatsCard } from '../../components/ui';
import type { WorkOrder } from '../../services/workOrdersApi';

// ─── Skeleton ────────────────────────────────────────────────────────

function TechnicianHomeSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-24 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        ))}
      </div>
      <div className="h-64 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      <div className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl" />
    </div>
  );
}

// ─── Helpers ─────────────────────────────────────────────────────────

function isOverdue(wo: WorkOrder): boolean {
  if (!wo.sla_deadline) return false;
  return new Date(wo.sla_deadline) < new Date();
}

// ─── Main Component ──────────────────────────────────────────────────

export function TechnicianHome() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { data: workOrders, isLoading } = useWorkOrders();

  const allOrders = useMemo(() => {
    if (!workOrders) return [];
    return Array.isArray(workOrders) ? workOrders : [];
  }, [workOrders]);

  // My tasks: today's assigned + in_progress
  const myTasks = useMemo(() => {
    return allOrders.filter(
      (wo) =>
        wo.assigned_to === user?.id &&
        (wo.status === 'open' || wo.status === 'in_progress'),
    );
  }, [allOrders, user?.id]);

  // Overdue tasks
  const overdueTasks = useMemo(() => {
    return allOrders.filter(
      (wo) =>
        wo.assigned_to === user?.id &&
        (wo.status === 'open' || wo.status === 'in_progress') &&
        isOverdue(wo),
    );
  }, [allOrders, user?.id]);

  if (isLoading) {
    return <TechnicianHomeSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatsCard
          title={t('my_tasks') || 'My Tasks'}
          value={myTasks.length}
          icon={ClipboardList}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('overdue') || 'Overdue'}
          value={overdueTasks.length}
          icon={AlertTriangle}
          iconBgColor="bg-red-50"
          iconColor={overdueTasks.length > 0 ? 'text-red-600' : 'text-slate-400'}
        />
        <StatsCard
          title={t('completed_today') || 'Completed Today'}
          value={
            allOrders.filter(
              (wo) =>
                wo.assigned_to === user?.id && wo.status === 'completed',
            ).length
          }
          icon={CheckCircle2}
          iconBgColor="bg-emerald-50"
          iconColor="text-emerald-600"
        />
      </div>

      {/* Quick QR Scan */}
      <Card>
        <CardBody>
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                {t('quick_qr_scan') || 'Quick QR Scan'}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                {t('scan_qr_to_open_work_order') || 'Scan QR code to open work order'}
              </p>
            </div>
            <Button
              variant="primary"
              icon={<QrCode className="w-5 h-5" />}
              onClick={() => navigate('/work-orders')}
            >
              {t('scan') || 'Scan'}
            </Button>
          </div>
        </CardBody>
      </Card>

      {/* My Tasks (Today) */}
      <Card>
        <CardHeader
          action={
            <Button
              variant="ghost"
              size="sm"
              icon={<ArrowRight className="w-4 h-4" />}
              onClick={() => navigate('/work-orders')}
            >
              {t('view_all') || 'View All'}
            </Button>
          }
        >
          {t('my_tasks_today') || "Today's Tasks"}
        </CardHeader>
        <CardBody>
          {myTasks.length === 0 ? (
            <div className="text-center py-8">
              <ClipboardList className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
              <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                {t('no_tasks_assigned') || 'No tasks assigned'}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                {t('no_tasks_description') || 'You have no work orders for today'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {myTasks.slice(0, 10).map((wo) => (
                <div
                  key={wo.id}
                  onClick={() => navigate(`/work-orders/${wo.id}`)}
                  className="flex items-center justify-between p-3 rounded-lg bg-slate-50 dark:bg-slate-900/30 hover:bg-slate-100 dark:hover:bg-slate-800/70 cursor-pointer transition-colors border border-transparent dark:border-slate-800"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <Clock className="w-4 h-4 text-slate-400 shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
                        {wo.device_name || wo.device_id || wo.id.slice(0, 8)}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                        {wo.device_id} {wo.sla_deadline ? `• ${new Date(wo.sla_deadline).toLocaleDateString()}` : ''}
                      </p>
                    </div>
                  </div>
                  <Badge
                    variant={
                      wo.priority === 'critical'
                        ? 'danger'
                        : wo.priority === 'high'
                          ? 'warning'
                          : wo.priority === 'medium'
                            ? 'info'
                            : 'success'
                    }
                    size="sm"
                  >
                    {wo.priority}
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>

      {/* Overdue Tasks */}
      {overdueTasks.length > 0 && (
        <Card>
          <CardHeader>
            <span className="flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-red-500" />
              {t('overdue_tasks') || 'Overdue Tasks'}
            </span>
          </CardHeader>
          <CardBody>
            <div className="space-y-2">
              {overdueTasks.slice(0, 5).map((wo) => (
                <div
                  key={wo.id}
                  onClick={() => navigate(`/work-orders/${wo.id}`)}
                  className="flex items-center justify-between p-3 rounded-lg bg-red-50 dark:bg-red-900/20 hover:bg-red-100 dark:hover:bg-red-900/30 cursor-pointer transition-colors border border-red-200 dark:border-red-800/50"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <AlertTriangle className="w-4 h-4 text-red-500 shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-red-800 dark:text-red-300 truncate">
                        {wo.device_name || wo.device_id || wo.id.slice(0, 8)}
                      </p>
                      <p className="text-xs text-red-600 dark:text-red-400">
                        {t('sla_breached') || 'SLA Breached'}{' '}
                        {wo.sla_deadline
                          ? `• ${new Date(wo.sla_deadline).toLocaleDateString()}`
                          : ''}
                      </p>
                    </div>
                  </div>
                  <Badge variant="danger" size="sm">
                    {wo.priority}
                  </Badge>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}

export default TechnicianHome;
