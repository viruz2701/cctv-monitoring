// ═══════════════════════════════════════════════════════════════════════
// AssetExplorer — Asset tree drill-down page
// UX-4.1: Asset Tree Drill-down (feature flag: asset_tree_drilldown)
//   - Hierarchy: Organization → Site → Building → Floor → Device → Component
//   - Sticky left panel with asset tree
//   - Context in URL through searchParams
//   - Lazy loading for large trees
//   - Search within tree
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useMemo, Suspense } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { isFeatureEnabled } from '../config/featureFlags';
import {
  HardDrive,
  Search,
  RefreshCw,
  Loader2,
  MapPin,
  Layers,
  Monitor,
} from '../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, StatsCard, Breadcrumbs } from '../components/ui';
import { AssetTree, type AssetTreeNode, type BreadcrumbItem } from '../components/organisms/AssetTree';
import type { AssetNodeType } from '../components/organisms/AssetTree';

// ─── Skeleton ────────────────────────────────────────────────────────

function AssetExplorerSkeleton() {
  return (
    <div className="flex gap-6 animate-pulse">
      <div className="w-80 shrink-0 space-y-3">
        <div className="h-10 bg-slate-200 dark:bg-slate-700 rounded-lg" />
        <div className="h-96 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      </div>
      <div className="flex-1 space-y-4">
        <div className="h-8 bg-slate-200 dark:bg-slate-700 rounded-lg w-1/3" />
        <div className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-24 bg-slate-200 dark:bg-slate-700 rounded-xl" />
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Detail Panel ────────────────────────────────────────────────────

interface DetailPanelProps {
  node: AssetTreeNode | null;
  breadcrumbs: BreadcrumbItem[];
  onNavigate: (nodeId: string) => void;
}

function DetailPanel({ node, breadcrumbs, onNavigate }: DetailPanelProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  if (!node) {
    return (
      <div className="flex flex-col items-center justify-center h-96 text-slate-400">
        <Layers className="w-16 h-16 mb-4 text-slate-300" />
        <p className="text-sm font-medium">{t('select_asset') || 'Select an asset node'}</p>
        <p className="text-xs mt-1">
          {t('select_asset_description') || 'Choose an item from the tree to view details'}
        </p>
      </div>
    );
  }

  const typeLabelMap: Record<AssetNodeType, string> = {
    organization: t('organization') || 'Organization',
    site: t('site') || 'Site',
    building: t('building') || 'Building',
    floor: t('floor') || 'Floor',
    room: t('room') || 'Room',
    device: t('device') || 'Device',
  };

  return (
    <div className="space-y-4">
      {/* Breadcrumb path */}
      <div className="flex items-center gap-1.5 text-xs text-slate-500 flex-wrap">
        {breadcrumbs.map((crumb, idx) => {
          const isLast = idx === breadcrumbs.length - 1;
          return (
            <React.Fragment key={crumb.id}>
              {idx > 0 && <span className="text-slate-300">/</span>}
              <button
                onClick={() => onNavigate(crumb.id)}
                className={`hover:underline ${
                  isLast
                    ? 'text-blue-600 font-medium'
                    : 'text-slate-500 hover:text-slate-700'
                }`}
              >
                {crumb.name}
              </button>
            </React.Fragment>
          );
        })}
      </div>

      {/* Node Details */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <HardDrive className="w-5 h-5 text-slate-500" />
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">{node.name}</h2>
          </div>
        </CardHeader>
        <CardBody>
          <div className="space-y-3">
            <div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800">
              <span className="text-sm text-slate-500">{t('type') || 'Type'}</span>
              <span className="text-sm font-medium text-slate-900 dark:text-white">
                {typeLabelMap[node.type] || node.type}
              </span>
            </div>
            <div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800">
              <span className="text-sm text-slate-500">{t('status') || 'Status'}</span>
              <span
                className={`text-sm font-medium capitalize ${
                  node.status === 'active' || node.status === 'ONLINE'
                    ? 'text-emerald-600'
                    : node.status === 'inactive' || node.status === 'OFFLINE'
                      ? 'text-red-600'
                      : 'text-amber-600'
                }`}
              >
                {node.status}
              </span>
            </div>
            <div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800">
              <span className="text-sm text-slate-500">{t('devices') || 'Devices'}</span>
              <span className="text-sm font-medium text-slate-900 dark:text-white">
                {node.deviceCount}
              </span>
            </div>
            <div className="flex justify-between py-2">
              <span className="text-sm text-slate-500">{t('children') || 'Children'}</span>
              <span className="text-sm font-medium text-slate-900 dark:text-white">
                {node.children.length}
              </span>
            </div>
          </div>
        </CardBody>
      </Card>

      {/* Device Quick Stats */}
      {node.devices && node.devices.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <StatsCard
            title={t('total_devices') || 'Total Devices'}
            value={node.devices.length}
            icon={HardDrive}
            iconBgColor="bg-indigo-50"
            iconColor="text-indigo-600"
          />
          <StatsCard
            title={t('online') || 'Online'}
            value={node.devices.filter((d) => d.status === 'ONLINE').length}
            icon={Monitor}
            iconBgColor="bg-emerald-50"
            iconColor="text-emerald-600"
          />
          <StatsCard
            title={t('offline') || 'Offline'}
            value={node.devices.filter((d) => d.status === 'OFFLINE').length}
            icon={MapPin}
            iconBgColor="bg-red-50"
            iconColor="text-red-600"
          />
        </div>
      )}

      {/* Navigate to device list */}
      {node.deviceCount > 0 && (
        <Button
          variant="primary"
          onClick={() => {
            if (node.type === 'device' && node.devices?.[0]) {
              navigate(`/devices/${node.devices[0].device_id}`);
            } else {
              navigate(`/devices?site_id=${node.id}`);
            }
          }}
        >
          {t('view_devices') || 'View Devices'}
        </Button>
      )}
    </div>
  );
}

// ─── Main Component ──────────────────────────────────────────────────

export function AssetExplorer() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  // URL state: ?node=node-id
  const selectedNodeId = searchParams.get('node') || null;

  // Asset tree state
  const [selectedNode, setSelectedNode] = useState<AssetTreeNode | null>(null);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([]);
  const [treeKey, setTreeKey] = useState(0);

  const handleNodeSelect = useCallback(
    (node: AssetTreeNode, crumbs: BreadcrumbItem[]) => {
      setSelectedNode(node);
      setBreadcrumbs(crumbs);
      // Sync to URL
      setSearchParams({ node: node.id }, { replace: true });
    },
    [setSearchParams],
  );

  const handleNavigate = useCallback(
    (nodeId: string) => {
      setSearchParams({ node: nodeId }, { replace: true });
    },
    [setSearchParams],
  );

  const handleRefresh = useCallback(() => {
    setTreeKey((prev) => prev + 1);
    setSelectedNode(null);
    setBreadcrumbs([]);
    setSearchParams({}, { replace: true });
  }, [setSearchParams]);

  return (
    <div className="p-4 md:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('asset_explorer') || 'Asset Explorer'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('asset_explorer_description') || 'Drill into your asset hierarchy'}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          icon={<RefreshCw className="w-4 h-4" />}
          onClick={handleRefresh}
        >
          {t('refresh') || 'Refresh'}
        </Button>
      </div>

      {/* Two-column layout: Tree (sticky left) + Detail */}
      <div className="flex flex-col lg:flex-row gap-6">
        {/* Left Panel — Asset Tree (sticky) */}
        <div className="w-full lg:w-80 shrink-0">
          <div className="lg:sticky lg:top-4 space-y-4">
            <Card>
              <CardBody className="p-0">
                <div className="max-h-[calc(100vh-12rem)] overflow-y-auto">
                  <Suspense
                    fallback={
                      <div className="flex items-center justify-center py-12">
                        <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
                      </div>
                    }
                  >
                    <AssetTree
                      key={treeKey}
                      onNodeSelect={handleNodeSelect}
                      hideStats
                    />
                  </Suspense>
                </div>
              </CardBody>
            </Card>
          </div>
        </div>

        {/* Right Panel — Detail */}
        <div className="flex-1 min-w-0">
          <DetailPanel
            node={selectedNode}
            breadcrumbs={breadcrumbs}
            onNavigate={handleNavigate}
          />
        </div>
      </div>
    </div>
  );
}

export default AssetExplorer;
