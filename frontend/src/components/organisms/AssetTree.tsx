// ═══════════════════════════════════════════════════════════════════════
// AssetTree — hierarchical asset tree (P2-2)
// Hierarchy: Organization → Site → Building → Floor → Room → Device
// Based on LocationTree.tsx but extended with:
//   - Extended hierarchy levels
//   - Animated expand/collapse
//   - Breadcrumb path tracking
//   - Per-level icons (Building2, MapPin, Layers, etc.)
//   - Device count badge on each node
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  ChevronRight,
  ChevronDown,
  Building2,
  MapPin,
  Layers,
  Monitor,
  Camera,
  HardDrive,
  Search,
  RefreshCw,
  Expand,
  Globe,
  Home,
  Server,
} from '../ui/Icons';
import { Card, Badge, Button, StatsCard } from '../ui';

// ── Types ────────────────────────────────────────────────────────────

export type AssetNodeType =
  | 'organization'
  | 'site'
  | 'building'
  | 'floor'
  | 'room'
  | 'device';

export interface AssetDeviceInfo {
  device_id: string;
  name: string;
  device_type: string;
  status: string;
}

export interface AssetTreeNode {
  id: string;
  name: string;
  type: AssetNodeType;
  status: string;
  children: AssetTreeNode[];
  deviceCount: number;
  devices?: AssetDeviceInfo[];
  parent_id?: string;
  level: number;
  collapsed?: boolean;
}

export interface BreadcrumbItem {
  id: string;
  name: string;
  type: AssetNodeType;
}

// ── Icons ────────────────────────────────────────────────────────────

const TYPE_ICONS: Record<AssetNodeType, React.FC<any>> = {
  organization: Globe,
  site: Building2,
  building: Home,
  floor: Layers,
  room: Monitor,
  device: Camera,
};

const DEFAULT_ICON = MapPin;

// ── Props ────────────────────────────────────────────────────────────

interface AssetTreeProps {
  /** External data override (if not provided, fetches internally) */
  data?: AssetTreeNode[];
  /** Loading state override */
  loading?: boolean;
  /** Error message override */
  error?: string | null;
  /** Callback when a device is clicked */
  onDeviceClick?: (deviceId: string) => void;
  /** Callback when a node is selected (for breadcrumb tracking) */
  onNodeSelect?: (node: AssetTreeNode, breadcrumbs: BreadcrumbItem[]) => void;
  /** Hide stats cards */
  hideStats?: boolean;
  /** Hide search input */
  hideSearch?: boolean;
  /** Initial expand state */
  defaultExpandAll?: boolean;
}

// ── Animated Expand/Collapse ─────────────────────────────────────────

function AnimatedCollapse({ children, isOpen }: { children: React.ReactNode; isOpen: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState<number | undefined>(isOpen ? undefined : 0);

  useEffect(() => {
    if (!ref.current) return;
    const el = ref.current;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const scrollH = entry.contentRect.height;
        if (isOpen) {
          setHeight(scrollH);
        }
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [isOpen, children]);

  useEffect(() => {
    if (isOpen) {
      const el = ref.current;
      if (el) {
        // Trigger expand animation
        const scrollH = el.scrollHeight;
        setHeight(scrollH);
      }
    } else {
      setHeight(0);
    }
  }, [isOpen, children]);

  return (
    <div
      className="overflow-hidden transition-all duration-300 ease-in-out"
      style={{ height: height !== undefined ? `${height}px` : 'auto' }}
    >
      <div ref={ref}>{children}</div>
    </div>
  );
}

// ── Tree Node Component ──────────────────────────────────────────────

function AssetTreeNode({
  node,
  onToggle,
  onDeviceClick,
  searchTerm,
  breadcrumbs,
  onNodeSelect,
  focusedNodeId,
  onFocusChange,
}: {
  node: AssetTreeNode;
  onToggle: (id: string) => void;
  onDeviceClick: (deviceId: string) => void;
  searchTerm: string;
  breadcrumbs: BreadcrumbItem[];
  onNodeSelect?: (node: AssetTreeNode, crumbs: BreadcrumbItem[]) => void;
  focusedNodeId?: string | null;
  onFocusChange?: (id: string | null) => void;
}) {
  const Icon = TYPE_ICONS[node.type] || DEFAULT_ICON;
  const hasChildren = node.children.length > 0 || (node.devices && node.devices.length > 0);
  const isExpanded = !node.collapsed;
  const isHighlighted =
    searchTerm &&
    (node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      node.devices?.some((d) => d.name?.toLowerCase().includes(searchTerm.toLowerCase())));

  const statusColor =
    node.status === 'active' || node.status === 'ONLINE'
      ? 'text-emerald-600 bg-emerald-50'
      : node.status === 'inactive' || node.status === 'OFFLINE'
      ? 'text-slate-400 bg-slate-100'
      : 'text-amber-600 bg-amber-50';

  const nodeBreadcrumbs = useMemo(
    () => [...breadcrumbs, { id: node.id, name: node.name, type: node.type }],
    [breadcrumbs, node.id, node.name, node.type]
  );

  const handleClick = useCallback(() => {
    onToggle(node.id);
    onNodeSelect?.(node, nodeBreadcrumbs);
  }, [node, nodeBreadcrumbs, onToggle, onNodeSelect]);

  const isFocused = focusedNodeId === node.id;
  const nodeRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Let the tree container handle arrow navigation
    if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Enter', ' '].includes(e.key)) {
      return; // handled by the container's onKeyDown
    }
  }, []);

  const handleFocus = useCallback(() => {
    onFocusChange?.(node.id);
  }, [node.id, onFocusChange]);

  return (
    <div>
      <div
        ref={nodeRef}
        role="treeitem"
        tabIndex={isFocused ? 0 : -1}
        aria-expanded={hasChildren ? isExpanded : undefined}
        aria-selected={isFocused}
        data-node-id={node.id}
        className={`flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors
          ${isHighlighted ? 'bg-yellow-50 border border-yellow-200' : 'hover:bg-slate-50'}
          ${isFocused ? 'ring-2 ring-blue-500 ring-inset' : ''}`}
        style={{ paddingLeft: `${12 + node.level * 24}px` }}
        onClick={handleClick}
        onFocus={handleFocus}
        onKeyDown={handleKeyDown}
      >
        {/* Expand/collapse */}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggle(node.id);
          }}
          className="p-0.5 text-slate-400 hover:text-slate-600 transition-colors"
          aria-label={isExpanded ? 'Collapse' : 'Expand'}
        >
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronRight className="w-4 h-4" />
            )
          ) : (
            <span className="w-4 h-4 block" />
          )}
        </button>

        {/* Type icon */}
        <div className={`p-1 rounded-md ${
          node.type === 'organization' ? 'bg-purple-100 text-purple-600' :
          node.type === 'site' ? 'bg-blue-100 text-blue-600' :
          node.type === 'building' ? 'bg-indigo-100 text-indigo-600' :
          node.type === 'floor' ? 'bg-cyan-100 text-cyan-600' :
          node.type === 'room' ? 'bg-slate-100 text-slate-600' :
          'bg-slate-100 text-slate-500'
        }`}>
          <Icon className="w-3.5 h-3.5" />
        </div>

        {/* Name */}
        <span
          className={`text-sm flex-1 ${
            isHighlighted ? 'font-bold text-yellow-800' : 'font-medium text-slate-900'
          }`}
        >
          {node.name}
        </span>

        {/* Type label */}
        <span className="text-[10px] text-slate-400 uppercase tracking-wider hidden sm:block">
          {node.type}
        </span>

        {/* Status badge */}
        <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${statusColor}`}>
          {node.status}
        </span>

        {/* Device count badge */}
        {node.deviceCount > 0 && (
          <Badge variant="info" size="sm">
            {node.deviceCount} {node.deviceCount === 1 ? 'dev' : 'devs'}
          </Badge>
        )}
      </div>

      {/* Children with animation */}
      <AnimatedCollapse isOpen={isExpanded}>
        <div>
          {node.children.map((child) => (
            <AssetTreeNode
              key={child.id}
              node={child}
              onToggle={onToggle}
              onDeviceClick={onDeviceClick}
              searchTerm={searchTerm}
              breadcrumbs={nodeBreadcrumbs}
              onNodeSelect={onNodeSelect}
              focusedNodeId={focusedNodeId}
              onFocusChange={onFocusChange}
            />
          ))}
          {node.devices?.map((device) => {
            const DevIcon = TYPE_ICONS.device || Camera;
            const devStatusColor =
              device.status === 'ONLINE'
                ? 'text-emerald-600 bg-emerald-50'
                : device.status === 'OFFLINE'
                ? 'text-red-600 bg-red-50'
                : 'text-amber-600 bg-amber-50';
            const isDevHighlighted =
              searchTerm && device.name?.toLowerCase().includes(searchTerm.toLowerCase());

            return (
              <div
                key={device.device_id}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg cursor-pointer transition-colors
                  ${isDevHighlighted ? 'bg-yellow-50 border border-yellow-200' : 'hover:bg-slate-50'}`}
                style={{ paddingLeft: `${36 + (node.level + 1) * 24}px` }}
                onClick={() => onDeviceClick(device.device_id)}
              >
                <span className="w-4 h-4 block" />
                <div className="p-1 rounded-md bg-slate-100 text-slate-500">
                  <DevIcon className="w-3 h-3" />
                </div>
                <span
                  className={`text-xs flex-1 ${
                    isDevHighlighted ? 'font-bold text-yellow-800' : 'text-slate-700'
                  }`}
                >
                  {device.name || device.device_id}
                </span>
                <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${devStatusColor}`}>
                  {device.status}
                </span>
              </div>
            );
          })}
        </div>
      </AnimatedCollapse>
    </div>
  );
}

// ── Main Component ───────────────────────────────────────────────────

export function AssetTree({
  data: externalData,
  loading: externalLoading,
  error: externalError,
  onDeviceClick: externalDeviceClick,
  onNodeSelect,
  hideStats = false,
  hideSearch = false,
  defaultExpandAll = false,
}: AssetTreeProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [treeData, setTreeData] = useState<AssetTreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandAll, setExpandAll] = useState(defaultExpandAll);
  const [stats, setStats] = useState({ organizations: 0, sites: 0, devices: 0, online: 0, offline: 0 });
  const [error, setError] = useState<string | null>(null);
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([]);

  // Use external data if provided
  const isExternal = !!externalData;
  const effectiveLoading = isExternal ? (externalLoading ?? false) : loading;
  const effectiveError = isExternal ? (externalError ?? null) : error;
  const effectiveData = isExternal ? externalData : treeData;

  const fetchData = useCallback(async () => {
    if (isExternal) return;
    setLoading(true);
    setError(null);
    try {
      // Dynamic import to avoid circular deps
      const { request } = await import('../../services/api');
      const [sitesData, devicesData] = await Promise.all([
        request<any[]>('/sites'),
        request<any[]>('/devices'),
      ]);

      const sites = sitesData || [];
      const devices = devicesData || [];

      // Build tree: Organization → Site → Building → Floor → Room
      const orgMap = new Map<string, any>();
      const uncategorizedOrg = '__default__';

      // Group sites by organization
      for (const site of sites) {
        const orgName = site.organization || 'Uncategorized';
        if (!orgMap.has(orgName)) {
          orgMap.set(orgName, []);
        }
        orgMap.get(orgName)!.push(site);
      }

      // Build device map
      const deviceMap = new Map<string, AssetDeviceInfo[]>();
      for (const device of devices) {
        const siteId = device.site_id || '';
        if (!deviceMap.has(siteId)) deviceMap.set(siteId, []);
        deviceMap.get(siteId)!.push({
          device_id: device.device_id,
          name: device.name || device.device_id,
          device_type: device.vendor_type || 'camera',
          status: device.status || 'UNKNOWN',
        });
      }

      // Build hierarchy
      let level = 0;
      const rootNodes: AssetTreeNode[] = [];

      for (const [orgName, orgSites] of orgMap.entries()) {
        const orgNode: AssetTreeNode = {
          id: `org-${orgName.replace(/\s+/g, '-').toLowerCase()}`,
          name: orgName,
          type: 'organization',
          status: 'active',
          children: [],
          deviceCount: 0,
          level,
          collapsed: !expandAll,
        };

        for (const site of orgSites) {
          const siteDevices = deviceMap.get(site.id) || [];
          const siteNode: AssetTreeNode = {
            id: site.id,
            name: site.name || site.id,
            type: 'site',
            status: site.status || 'active',
            children: [],
            deviceCount: siteDevices.length,
            devices: siteDevices,
            level: level + 1,
            collapsed: !expandAll,
          };

          // Add buildings/levels if the site has address parts
          if (site.address) {
            const addressParts = site.address.split(',').map((p: string) => p.trim()).filter(Boolean);
            for (let i = 0; i < Math.min(addressParts.length, 3); i++) {
              const subNode: AssetTreeNode = {
                id: `${site.id}-level-${i}`,
                name: addressParts[i],
                type: i === 0 ? 'building' : i === 1 ? 'floor' : 'room',
                status: 'active',
                children: [],
                deviceCount: i === addressParts.length - 1 ? siteDevices.length : 0,
                devices: i === addressParts.length - 1 ? siteDevices : undefined,
                level: level + 2 + i,
                collapsed: !expandAll,
              };
              siteNode.children.push(subNode);
              siteNode.deviceCount += subNode.deviceCount;
            }
          }

          // If no children but has devices, attach devices directly
          if (siteNode.children.length === 0 && siteDevices.length > 0) {
            const roomNode: AssetTreeNode = {
              id: `${site.id}-devices`,
              name: 'Devices',
              type: 'room',
              status: 'active',
              children: [],
              deviceCount: siteDevices.length,
              devices: siteDevices,
              level: level + 2,
              collapsed: !expandAll,
            };
            siteNode.children.push(roomNode);
          }

          orgNode.children.push(siteNode);
          orgNode.deviceCount += siteNode.deviceCount;
        }

        rootNodes.push(orgNode);
      }

      setTreeData(rootNodes);

      // Stats
      const online = devices.filter((d: any) => d.status === 'ONLINE').length;
      const offline = devices.filter((d: any) => d.status === 'OFFLINE').length;
      setStats({
        organizations: orgMap.size,
        sites: sites.length,
        devices: devices.length,
        online,
        offline,
      });
    } catch (err: any) {
      setError(err.message || 'Failed to load asset data');
    } finally {
      setLoading(false);
    }
  }, [expandAll, isExternal]);

  useEffect(() => {
    if (!isExternal) {
      fetchData();
    }
  }, []);

  const handleDeviceClick = useCallback(
    (deviceId: string) => {
      if (externalDeviceClick) {
        externalDeviceClick(deviceId);
      } else {
        navigate(`/devices/${deviceId}`);
      }
    },
    [externalDeviceClick, navigate]
  );

  const toggleNode = useCallback((id: string) => {
    setTreeData((prev) => toggleNodeInTree(prev, id));
  }, []);

  const toggleAll = useCallback(() => {
    setExpandAll((prev) => !prev);
  }, []);

  const handleNodeSelect = useCallback(
    (node: AssetTreeNode, crumbs: BreadcrumbItem[]) => {
      setBreadcrumbs(crumbs);
      onNodeSelect?.(node, crumbs);
    },
    [onNodeSelect]
  );

  // ── Keyboard Navigation (WCAG 2.1 AA) ────────────────────────────

  const [focusedNodeId, setFocusedNodeId] = useState<string | null>(null);
  const treeContainerRef = useRef<HTMLDivElement>(null);

  const handleTreeKeyDown = useCallback((e: React.KeyboardEvent) => {
    const tree = treeContainerRef.current;
    if (!tree) return;

    const items = tree.querySelectorAll<HTMLElement>('[role="treeitem"]');
    if (items.length === 0) return;

    const currentIndex = focusedNodeId
      ? Array.from(items).findIndex((el) => el.dataset.nodeId === focusedNodeId)
      : -1;

    switch (e.key) {
      case 'ArrowDown': {
        e.preventDefault();
        const next = Math.min(currentIndex + 1, items.length - 1);
        items[next]?.focus();
        setFocusedNodeId(items[next]?.dataset.nodeId ?? null);
        break;
      }
      case 'ArrowUp': {
        e.preventDefault();
        const prev = Math.max(currentIndex - 1, 0);
        items[prev]?.focus();
        setFocusedNodeId(items[prev]?.dataset.nodeId ?? null);
        break;
      }
      case 'ArrowRight': {
        e.preventDefault();
        if (currentIndex >= 0) {
          const el = items[currentIndex];
          const expandBtn = el?.querySelector<HTMLElement>('[aria-label="Expand"], [aria-label="Collapse"]');
          if (expandBtn && el?.getAttribute('aria-expanded') === 'false') {
            expandBtn.click();
          }
        }
        break;
      }
      case 'ArrowLeft': {
        e.preventDefault();
        if (currentIndex >= 0) {
          const el = items[currentIndex];
          const expandBtn = el?.querySelector<HTMLElement>('[aria-label="Expand"], [aria-label="Collapse"]');
          if (expandBtn && el?.getAttribute('aria-expanded') === 'true') {
            expandBtn.click();
          }
        }
        break;
      }
      case 'Enter':
      case ' ': {
        e.preventDefault();
        if (currentIndex >= 0) {
          items[currentIndex]?.click();
        }
        break;
      }
    }
  }, [focusedNodeId]);

  // ── Filter by search ──────────────────────────────────────────────

  const filteredTree = useMemo(() => {
    if (!searchTerm) return effectiveData;
    return filterTree(effectiveData, searchTerm.toLowerCase());
  }, [effectiveData, searchTerm]);

  return (
    <div className="space-y-4">
      {/* Stats */}
      {!hideStats && !isExternal && (
        <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
          <StatsCard
            title={t('organizations') || 'Organizations'}
            value={stats.organizations}
            icon={Globe}
            iconBgColor="bg-purple-50"
            iconColor="text-purple-600"
          />
          <StatsCard
            title={t('sites') || 'Sites'}
            value={stats.sites}
            icon={Building2}
            iconBgColor="bg-blue-50"
            iconColor="text-blue-600"
          />
          <StatsCard
            title={t('devices') || 'Devices'}
            value={stats.devices}
            icon={HardDrive}
            iconBgColor="bg-indigo-50"
            iconColor="text-indigo-600"
          />
          <StatsCard
            title={t('online') || 'Online'}
            value={stats.online}
            icon={Camera}
            iconBgColor="bg-emerald-50"
            iconColor="text-emerald-600"
          />
          <StatsCard
            title={t('offline') || 'Offline'}
            value={stats.offline}
            icon={MapPin}
            iconBgColor="bg-red-50"
            iconColor="text-red-600"
          />
        </div>
      )}

      {/* Search + Actions */}
      {!hideSearch && (
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder={t('search_asset') || 'Search assets...'}
              className="w-full pl-9 pr-4 py-2 rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            icon={<Expand className="w-4 h-4" />}
            onClick={toggleAll}
          >
            {expandAll ? (t('collapse_all') || 'Collapse') : (t('expand_all') || 'Expand')}
          </Button>
          {!isExternal && (
            <Button
              variant="outline"
              size="sm"
              icon={<RefreshCw className="w-4 h-4" />}
              onClick={fetchData}
              loading={effectiveLoading}
            >
              {t('refresh') || 'Refresh'}
            </Button>
          )}
        </div>
      )}

      {/* Breadcrumbs */}
      {breadcrumbs.length > 0 && (
        <div className="flex items-center gap-1.5 text-xs text-slate-500 flex-wrap">
          {breadcrumbs.map((crumb, idx) => {
            const CrumbIcon = TYPE_ICONS[crumb.type] || DEFAULT_ICON;
            const isLast = idx === breadcrumbs.length - 1;
            return (
              <React.Fragment key={crumb.id}>
                {idx > 0 && <ChevronRight className="w-3 h-3 text-slate-300" />}
                <span
                  className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded ${
                    isLast
                      ? 'text-blue-600 bg-blue-50 font-medium'
                      : 'text-slate-500 hover:text-slate-700 cursor-pointer'
                  }`}
                  onClick={() => {
                    if (!isLast) {
                      // Navigate to this level
                    }
                  }}
                >
                  <CrumbIcon className="w-3 h-3" />
                  {crumb.name}
                </span>
              </React.Fragment>
            );
          })}
        </div>
      )}

      {/* Error */}
      {effectiveError && (
        <div className="p-3 bg-red-50 rounded-lg border border-red-200 text-sm text-red-700">
          {effectiveError}
          {!isExternal && (
            <button onClick={fetchData} className="ml-2 text-red-600 underline">
              {t('retry') || 'Retry'}
            </button>
          )}
        </div>
      )}

      {/* Tree */}
      <Card>
        <div className="p-4">
          {effectiveLoading && effectiveData.length === 0 ? (
            <div className="flex items-center justify-center py-16">
              <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
              <span className="ml-2 text-sm text-slate-500">
                {t('loading') || 'Loading...'}
              </span>
            </div>
          ) : filteredTree.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <MapPin className="w-12 h-12 text-slate-300 mb-4" />
              <p className="text-sm text-slate-500">
                {searchTerm
                  ? t('no_search_results') || 'Nothing found'
                  : t('no_assets') || 'No assets'}
              </p>
            </div>
          ) : (
            <div
              ref={treeContainerRef}
              role="tree"
              aria-label={t('asset_tree') || 'Asset tree'}
              onKeyDown={handleTreeKeyDown}
              className="divide-y divide-slate-100"
            >
              {filteredTree.map((node) => (
                <AssetTreeNode
                  key={node.id}
                  node={node}
                  onToggle={toggleNode}
                  onDeviceClick={handleDeviceClick}
                  searchTerm={searchTerm}
                  breadcrumbs={[]}
                  onNodeSelect={handleNodeSelect}
                  focusedNodeId={focusedNodeId}
                  onFocusChange={setFocusedNodeId}
                />
              ))}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}

// ── Tree Utilities ───────────────────────────────────────────────────

function toggleNodeInTree(nodes: AssetTreeNode[], id: string): AssetTreeNode[] {
  return nodes.map((node) => {
    if (node.id === id) {
      return { ...node, collapsed: !node.collapsed };
    }
    if (node.children.length > 0) {
      return { ...node, children: toggleNodeInTree(node.children, id) };
    }
    return node;
  });
}

function filterTree(nodes: AssetTreeNode[], term: string): AssetTreeNode[] {
  return nodes
    .map((node) => {
      const nameMatch = node.name.toLowerCase().includes(term);
      const deviceMatch = node.devices?.some((d) => d.name?.toLowerCase().includes(term));
      const filteredChildren = filterTree(node.children, term);

      if (nameMatch || deviceMatch || filteredChildren.length > 0) {
        return {
          ...node,
          collapsed: false,
          children: filteredChildren,
        };
      }
      return null;
    })
    .filter(Boolean) as AssetTreeNode[];
}
