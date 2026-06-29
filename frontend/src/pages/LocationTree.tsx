import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Badge, Button, StatsCard } from '../components/ui';
import { useNavigate } from 'react-router-dom';
import {
  ChevronRight, ChevronDown, Building2, MapPin,
  HardDrive, Camera, Server, Monitor, Wifi,
  Search, RefreshCw, Layers, Home, Expand,
} from '../components/ui/Icons';

// ── Types ────────────────────────────────────────────────────────────

interface SiteNode {
  id: string;
  name: string;
  type: 'site' | 'building' | 'floor' | 'room' | 'rack';
  status: string;
  children: SiteNode[];
  deviceCount: number;
  devices?: DeviceInfo[];
  parent_id?: string;
  level: number;
  collapsed?: boolean;
}

interface DeviceInfo {
  device_id: string;
  name: string;
  device_type: string;
  status: string;
}

// ── Icons ────────────────────────────────────────────────────────────

const TYPE_ICONS: Record<string, React.FC<any>> = {
  site: Building2,
  building: Building2,
  floor: Layers,
  room: Monitor,
  rack: Server,
  switch: Wifi,
  nvr: Server,
  dvr: Server,
  camera: Camera,
  server: Server,
  encoder: Monitor,
};

const DEFAULT_ICON = MapPin;

// ── Tree Node Component ──────────────────────────────────────────────

function TreeNode({
  node,
  onToggle,
  onDeviceClick,
  searchTerm,
}: {
  node: SiteNode;
  onToggle: (id: string) => void;
  onDeviceClick: (deviceId: string) => void;
  searchTerm: string;
}) {
  const Icon = TYPE_ICONS[node.type] || DEFAULT_ICON;
  const hasChildren = node.children.length > 0 || (node.devices && node.devices.length > 0);
  const isExpanded = !node.collapsed;
  const isHighlighted = searchTerm && node.name.toLowerCase().includes(searchTerm.toLowerCase());

  const statusColor = node.status === 'active' ? 'text-emerald-600 bg-emerald-50' :
    node.status === 'inactive' ? 'text-slate-400 bg-slate-100' :
    'text-amber-600 bg-amber-50';

  return (
    <div>
      <div
        className={`flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors
          ${isHighlighted ? 'bg-yellow-50 border border-yellow-200' : 'hover:bg-slate-50'}`}
        style={{ paddingLeft: `${12 + node.level * 24}px` }}
        onClick={() => onToggle(node.id)}
      >
        {/* Expand/collapse */}
        <button
          onClick={(e) => { e.stopPropagation(); onToggle(node.id); }}
          className="p-0.5 text-slate-400 hover:text-slate-600 transition-colors"
        >
          {hasChildren ? (
            isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />
          ) : (
            <span className="w-4 h-4 block" />
          )}
        </button>

        {/* Type icon */}
        <Icon className="w-4 h-4 text-slate-500" />

        {/* Name */}
        <span className={`text-sm flex-1 ${isHighlighted ? 'font-bold text-yellow-800' : 'font-medium text-slate-900'}`}>
          {node.name}
        </span>

        {/* Badge */}
        <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${statusColor}`}>
          {node.status}
        </span>

        {/* Device count */}
        {node.deviceCount > 0 && (
          <Badge variant="info" size="sm">{node.deviceCount} устр.</Badge>
        )}
      </div>

      {/* Children */}
      {isExpanded && (
        <div>
          {node.children.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              onToggle={onToggle}
              onDeviceClick={onDeviceClick}
              searchTerm={searchTerm}
            />
          ))}
          {node.devices?.map((device) => {
            const DevIcon = TYPE_ICONS[device.device_type] || DEFAULT_ICON;
            const devStatusColor = device.status === 'ONLINE' ? 'text-emerald-600 bg-emerald-50' :
              device.status === 'OFFLINE' ? 'text-red-600 bg-red-50' :
              'text-amber-600 bg-amber-50';
            const isDevHighlighted = searchTerm && device.name?.toLowerCase().includes(searchTerm.toLowerCase());

            return (
              <div
                key={device.device_id}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg cursor-pointer transition-colors
                  ${isDevHighlighted ? 'bg-yellow-50 border border-yellow-200' : 'hover:bg-slate-50'}`}
                style={{ paddingLeft: `${36 + (node.level + 1) * 24}px` }}
                onClick={() => onDeviceClick(device.device_id)}
              >
                <span className="w-4 h-4 block" />
                <DevIcon className="w-3.5 h-3.5 text-slate-400" />
                <span className={`text-xs flex-1 ${isDevHighlighted ? 'font-bold text-yellow-800' : 'text-slate-700'}`}>
                  {device.name || device.device_id}
                </span>
                <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${devStatusColor}`}>
                  {device.status}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ── Main Component ───────────────────────────────────────────────────

export function LocationTree() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [treeData, setTreeData] = useState<SiteNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandAll, setExpandAll] = useState(false);
  const [stats, setStats] = useState({ sites: 0, devices: 0, online: 0, offline: 0 });
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [sitesData, devicesData] = await Promise.all([
        request<any[]>('/sites'),
        request<any[]>('/devices'),
      ]);

      const sites = sitesData || [];
      const devices = devicesData || [];

      // Build tree
      const siteMap = new Map<string, any>();
      for (const site of sites) {
        siteMap.set(site.id, { ...site, children: [], devices: [] });
      }

      // Attach devices to sites
      const deviceMap = new Map<string, DeviceInfo[]>();
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

      // Build tree hierarchy
      const rootNodes: SiteNode[] = [];
      const buildTree = (parentId: string | null, level: number): SiteNode[] => {
        const nodes: SiteNode[] = [];
        for (const site of sites) {
          const parentLoc = (site as any).parent_location_id || null;
          if (parentLoc === parentId) {
            const children = buildTree(site.id, level + 1);
            const devs = deviceMap.get(site.id) || [];
            nodes.push({
              id: site.id,
              name: site.name || site.id,
              type: level === 0 ? 'site' : 'building',
              status: site.status || 'active',
              children,
              deviceCount: children.length + devs.length,
              devices: devs,
              level,
              collapsed: !expandAll,
            });
          }
        }
        return nodes;
      };

      const tree = buildTree(null, 0);
      setTreeData(tree);

      // Stats
      const online = devices.filter((d: any) => d.status === 'ONLINE').length;
      const offline = devices.filter((d: any) => d.status === 'OFFLINE').length;
      setStats({ sites: sites.length, devices: devices.length, online, offline });
    } catch (err: any) {
      setError(err.message || 'Failed to load location data');
    } finally {
      setLoading(false);
    }
  }, [expandAll]);

  useEffect(() => { fetchData(); }, []);

  const toggleNode = (id: string) => {
    setTreeData((prev) => toggleNodeInTree(prev, id));
  };

  const toggleAll = () => {
    setExpandAll(!expandAll);
  };

  // ── Filter by search ──────────────────────────────────────────────

  const filteredTree = searchTerm
    ? filterTree(treeData, searchTerm.toLowerCase())
    : treeData;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Building2 className="w-6 h-6" />
            {t('location_tree') || 'Дерево объектов'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('location_tree_desc') || 'Иерархическая структура объектов и устройств'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" icon={<Expand className="w-4 h-4" />} onClick={toggleAll}>
            {expandAll ? (t('collapse_all') || 'Свернуть') : (t('expand_all') || 'Развернуть')}
          </Button>
          <Button variant="outline" size="sm" icon={<RefreshCw className="w-4 h-4" />} onClick={fetchData} loading={loading}>
            {t('refresh') || 'Обновить'}
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard title={t('sites') || 'Объекты'} value={stats.sites} icon={Building2} iconBgColor="bg-blue-50" iconColor="text-blue-600" />
        <StatsCard title={t('devices') || 'Устройства'} value={stats.devices} icon={HardDrive} iconBgColor="bg-indigo-50" iconColor="text-indigo-600" />
        <StatsCard title={t('online') || 'Онлайн'} value={stats.online} icon={Wifi} iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('offline') || 'Офлайн'} value={stats.offline} icon={Camera} iconBgColor="bg-red-50" iconColor="text-red-600" />
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          placeholder={t('search_location') || 'Поиск по названию...'}
          className="w-full pl-9 pr-4 py-2 rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      {/* Error */}
      {error && (
        <div className="p-3 bg-red-50 rounded-lg border border-red-200 text-sm text-red-700">
          {error}
          <button onClick={fetchData} className="ml-2 text-red-600 underline">{t('retry') || 'Повтор'}</button>
        </div>
      )}

      {/* Tree */}
      <Card>
        <div className="p-4">
          {loading && treeData.length === 0 ? (
            <div className="flex items-center justify-center py-16">
              <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
              <span className="ml-2 text-sm text-slate-500">{t('loading') || 'Загрузка...'}</span>
            </div>
          ) : filteredTree.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <MapPin className="w-12 h-12 text-slate-300 mb-4" />
              <p className="text-sm text-slate-500">
                {searchTerm ? (t('no_search_results') || 'Ничего не найдено') : (t('no_sites') || 'Нет объектов')}
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {filteredTree.map((node) => (
                <TreeNode
                  key={node.id}
                  node={node}
                  onToggle={toggleNode}
                  onDeviceClick={(deviceId) => navigate(`/devices/${deviceId}`)}
                  searchTerm={searchTerm}
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

function toggleNodeInTree(nodes: SiteNode[], id: string): SiteNode[] {
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

function filterTree(nodes: SiteNode[], term: string): SiteNode[] {
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
    .filter(Boolean) as SiteNode[];
}
