// ═══════════════════════════════════════════════════════════════════════
// commandIndex — сервис индексации команд для Command Palette
// UX-5.1: Command Palette with Regulatory Awareness
//
// Feature Flag: command_palette_regulatory
//
// Особенности:
//   - Contextual actions based on current page, user role, region
//   - Natural language → structured command mapping
//   - Fuzzy search with fuse.js
//   - Recent actions (last 5)
// ═══════════════════════════════════════════════════════════════════════

import Fuse, { type IFuseOptions } from 'fuse.js';

// ── Types ────────────────────────────────────────────────────────────

/** Regulatory region identifiers */
export type RegulatoryRegion = 'BY' | 'RU' | 'EU' | 'TR' | 'VN' | 'INTL';

/** Available action types for contextual commands */
export type ActionType =
  | 'navigate'
  | 'create_wo'
  | 'create_ticket'
  | 'open_compliance'
  | 'run_report'
  | 'toggle_setting'
  | 'view_device'
  | 'view_site'
  | 'quick_action';

/** User role hierarchy (higher index = more privileged) */
export type UserRole = 'viewer' | 'technician' | 'manager' | 'owner' | 'support' | 'admin';

export interface IndexedCommand {
  /** Unique command ID */
  id: string;
  /** Display label */
  label: string;
  /** Description / subtitle */
  description?: string;
  /** Search keywords (region-aware) */
  keywords: string[];
  /** Page path or action ID */
  path?: string;
  /** Action handler (for non-navigation commands) */
  action?: () => void;
  /** Required minimum role */
  minRole: UserRole;
  /** Applicable regulatory regions (empty = all regions) */
  regions?: RegulatoryRegion[];
  /** Page patterns where this command is relevant (regex) */
  contextPatterns?: string[];
  /** Category for grouping */
  category: string;
  /** Icon name (from lucide-react) */
  iconName: string;
  /** Whether this command should be shown in recent */
  trackable?: boolean;
}

export interface CommandContext {
  /** Current page path */
  currentPath: string;
  /** User role */
  role: UserRole;
  /** Regulatory region (from settings) */
  region: RegulatoryRegion;
  /** Recent command IDs (max 5) */
  recentCommandIds: string[];
}

// ── Fuse.js options ──────────────────────────────────────────────────

const FUSE_OPTIONS: IFuseOptions<IndexedCommand> = {
  keys: [
    { name: 'label', weight: 3 },
    { name: 'keywords', weight: 2 },
    { name: 'description', weight: 1 },
  ],
  threshold: 0.4,
  distance: 100,
  includeScore: true,
  minMatchCharLength: 1,
};

// ── All registered commands ──────────────────────────────────────────

const ALL_COMMANDS: IndexedCommand[] = [
  // ── Navigation ────────────────────────────────────────────────
  {
    id: 'nav-dashboard',
    label: 'Dashboard',
    description: 'Main health overview',
    keywords: ['home', 'overview', 'main', 'health', 'главная', 'дашборд'],
    path: '/dashboard',
    minRole: 'viewer',
    category: 'navigation',
    iconName: 'LayoutDashboard',
    trackable: true,
  },
  {
    id: 'nav-devices',
    label: 'Go to Devices',
    description: 'All CCTV devices',
    keywords: ['devices', 'camera', 'nvr', 'dvr', 'устройства', 'камеры'],
    path: '/devices',
    minRole: 'viewer',
    category: 'navigation',
    iconName: 'HardDrive',
    trackable: true,
  },
  {
    id: 'nav-work-orders',
    label: 'Work Orders',
    description: 'Manage maintenance tasks',
    keywords: ['work orders', 'wo', 'task', 'job', 'repair', 'наряды', 'то'],
    path: '/work-orders',
    minRole: 'technician',
    category: 'navigation',
    iconName: 'Ticket',
    trackable: true,
  },
  {
    id: 'nav-sites',
    label: 'Sites',
    description: 'Manage locations',
    keywords: ['sites', 'locations', 'address', 'building', 'объекты'],
    path: '/sites',
    minRole: 'viewer',
    category: 'navigation',
    iconName: 'MapPin',
    trackable: true,
  },
  {
    id: 'nav-compliance',
    label: 'Compliance Shield',
    description: 'Compliance risk assessment dashboard',
    keywords: ['compliance', 'gdpr', '152-fz', 'kvkk', 'pd', 'соответствие'],
    path: '/compliance-shield',
    minRole: 'manager',
    category: 'navigation',
    iconName: 'Shield',
    trackable: true,
  },

  // ── Regulatory actions ────────────────────────────────────────
  {
    id: 'action-compliance-by',
    label: 'Проверить compliance для BY',
    description: 'Open Belarus compliance dashboard',
    keywords: ['by', 'беларусь', 'рб', 'оац', 'приказ 66', 'stb', 'compliance belarus'],
    path: '/compliance-shield?region=BY',
    minRole: 'manager',
    regions: ['BY', 'RU'],
    category: 'regulatory',
    iconName: 'Shield',
    trackable: true,
  },
  {
    id: 'action-compliance-ru',
    label: 'Проверить compliance для РФ',
    description: '152-ФЗ, 149-ФЗ, KII compliance',
    keywords: ['ru', 'россия', 'рф', '152-фз', '149-фз', 'кии', 'фстэк', 'compliance russia'],
    path: '/compliance-shield?region=RU',
    minRole: 'manager',
    regions: ['BY', 'RU'],
    category: 'regulatory',
    iconName: 'Shield',
    trackable: true,
  },
  {
    id: 'action-compliance-gdpr',
    label: 'GDPR Compliance Check',
    description: 'EU data protection compliance',
    keywords: ['gdpr', 'eu', 'europe', 'data protection', 'dpia', 'nis2'],
    path: '/compliance-shield?region=EU',
    minRole: 'manager',
    regions: ['EU', 'INTL'],
    category: 'regulatory',
    iconName: 'Shield',
    trackable: true,
  },
  {
    id: 'action-compliance-kvkk',
    label: 'KVKK Compliance Check',
    description: 'Turkey data protection compliance',
    keywords: ['kvkk', 'turkey', 'türkiye', 'tr', 'verbis', 'personal data'],
    path: '/compliance-shield?region=TR',
    minRole: 'manager',
    regions: ['TR'],
    category: 'regulatory',
    iconName: 'Shield',
    trackable: true,
  },

  // ── Quick actions ─────────────────────────────────────────────
  {
    id: 'action-create-wo',
    label: 'Создать Work Order',
    description: 'Create new maintenance work order',
    keywords: ['create wo', 'new wo', 'создать то', 'новый наряд', 'maintenance', 'repair'],
    minRole: 'technician',
    category: 'actions',
    iconName: 'PlusCircle',
    trackable: true,
    contextPatterns: ['/devices', '/device/', '/work-orders', '/sites'],
  },
  {
    id: 'action-create-ticket',
    label: 'Создать Ticket',
    description: 'Create new support ticket',
    keywords: ['create ticket', 'new ticket', 'создать заявку', 'тикет', 'support'],
    minRole: 'technician',
    category: 'actions',
    iconName: 'PlusCircle',
    trackable: true,
    contextPatterns: ['/devices', '/device/', '/tickets'],
  },
  {
    id: 'action-run-report',
    label: 'Запустить отчет',
    description: 'Generate device health report',
    keywords: ['report', 'отчет', 'generate', 'export', 'health report'],
    path: '/reports',
    minRole: 'viewer',
    category: 'actions',
    iconName: 'FileText',
    trackable: true,
  },
  {
    id: 'action-toggle-dark',
    label: 'Переключить тему',
    description: 'Toggle dark/light mode',
    keywords: ['dark', 'light', 'theme', 'mode', 'тема', 'темная'],
    minRole: 'viewer',
    category: 'actions',
    iconName: 'SunMoon',
    trackable: false,
  },
  {
    id: 'action-shortcuts',
    label: 'Горячие клавиши',
    description: 'Show keyboard shortcuts',
    keywords: ['shortcuts', 'hotkeys', 'keyboard', 'клавиши', 'шорткаты'],
    minRole: 'viewer',
    category: 'actions',
    iconName: 'Keyboard',
    trackable: false,
  },

  // ── Device-specific contexts ──────────────────────────────────
  {
    id: 'context-device-history',
    label: 'История устройства',
    description: 'View device history timeline',
    keywords: ['history', 'timeline', 'events', 'alarms', 'история', 'события'],
    minRole: 'viewer',
    category: 'context',
    iconName: 'Clock',
    trackable: true,
    contextPatterns: ['/device/', '/devices/'],
  },
  {
    id: 'context-device-tunnel',
    label: 'Secure Tunnel',
    description: 'Open secure tunnel to device',
    keywords: ['tunnel', 'secure', 'ssh', 'vpn', 'туннель', 'безопасность'],
    minRole: 'technician',
    category: 'context',
    iconName: 'Shield',
    trackable: true,
    contextPatterns: ['/device/', '/devices/'],
  },
];

// ── Fuse Instance ────────────────────────────────────────────────────

let fuseInstance: Fuse<IndexedCommand> | null = null;

function getFuse(): Fuse<IndexedCommand> {
  if (!fuseInstance) {
    fuseInstance = new Fuse(ALL_COMMANDS, FUSE_OPTIONS);
  }
  return fuseInstance;
}

// ── Filtering helpers ────────────────────────────────────────────────

const ROLE_HIERARCHY: Record<UserRole, number> = {
  viewer: 0,
  technician: 1,
  manager: 2,
  owner: 3,
  support: 4,
  admin: 5,
};

function meetsMinRole(userRole: UserRole, minRole: UserRole): boolean {
  return ROLE_HIERARCHY[userRole] >= ROLE_HIERARCHY[minRole];
}

function matchesRegion(region: RegulatoryRegion, cmd: IndexedCommand): boolean {
  if (!cmd.regions || cmd.regions.length === 0) return true;
  return cmd.regions.includes(region);
}

function matchesContext(path: string, cmd: IndexedCommand): boolean {
  if (!cmd.contextPatterns || cmd.contextPatterns.length === 0) return true;
  return cmd.contextPatterns.some((pattern) => {
    try {
      return new RegExp(pattern).test(path);
    } catch {
      return path.startsWith(pattern);
    }
  });
}

// ── Public API ───────────────────────────────────────────────────────

/**
 * Get commands relevant to the given context.
 * Filters by role, region, and current page context.
 */
export function getContextualCommands(ctx: CommandContext): IndexedCommand[] {
  return ALL_COMMANDS.filter((cmd) => {
    if (!meetsMinRole(ctx.role, cmd.minRole)) return false;
    if (!matchesRegion(ctx.region, cmd)) return false;
    if (!matchesContext(ctx.currentPath, cmd)) return false;
    return true;
  });
}

/**
 * Search commands with fuse.js fuzzy search.
 * Returns results scoped to the given context.
 */
export function searchCommands(
  query: string,
  ctx: CommandContext,
  limit = 12,
): IndexedCommand[] {
  const contextual = getContextualCommands(ctx);

  if (!query.trim()) {
    // Return recent + contextual commands when no query
    const recents = ctx.recentCommandIds
      .map((id) => ALL_COMMANDS.find((c) => c.id === id))
      .filter((c): c is IndexedCommand => c != null && matchesRegion(ctx.region, c));

    const deduped = new Map<string, IndexedCommand>();
    for (const cmd of recents) {
      if (!deduped.has(cmd.id)) deduped.set(cmd.id, cmd);
    }
    for (const cmd of contextual) {
      if (!deduped.has(cmd.id)) deduped.set(cmd.id, cmd);
    }
    return Array.from(deduped.values()).slice(0, limit);
  }

  // Fuzzy search over the full command list, then contextual filter
  const fuse = getFuse();
  const raw = fuse.search(query);

  const filtered = raw
    .map((r) => r.item)
    .filter((cmd) => {
      if (!meetsMinRole(ctx.role, cmd.minRole)) return false;
      if (!matchesRegion(ctx.region, cmd)) return false;
      return true;
    });

  return filtered.slice(0, limit);
}

/**
 * Get a command by its ID.
 */
export function getCommandById(id: string): IndexedCommand | undefined {
  return ALL_COMMANDS.find((c) => c.id === id);
}

/**
 * Parse natural language input into a structured action.
 * Supports patterns like:
 *   - "создать ТО для камеры на складе" → Create WO
 *   - "проверить compliance для BY" → Open Compliance Dashboard
 */
export function parseNaturalLanguage(input: string): { commandId?: string; confidence: number } {
  const lower = input.toLowerCase().trim();

  // ── Create WO patterns ────────────────────────────────────────
  if (
    /^(созда(ть|й)|new|create)\s.*(то|wo|work.?order|наряд|ремонт|maintenance)/i.test(lower)
  ) {
    return { commandId: 'action-create-wo', confidence: 0.9 };
  }

  // ── Create Ticket patterns ────────────────────────────────────
  if (
    /^(созда(ть|й)|new|create)\s.*(ticket|тикет|заявк|типет)/i.test(lower)
  ) {
    return { commandId: 'action-create-ticket', confidence: 0.9 };
  }

  // ── Compliance check patterns ─────────────────────────────────
  const regionMatch = lower.match(/(compliance|соответствие|check)\s.*(by|беларус|рб|ru|росси|рф|eu|gdpr|tr|turkey|kvkk)/i);
  if (regionMatch) {
    const regionStr = regionMatch[1].toUpperCase();
    const regionMap: Record<string, RegulatoryRegion> = {
      BY: 'BY', БЕЛАРУС: 'BY', РБ: 'BY',
      RU: 'RU', РОССИ: 'RU', РФ: 'RU',
      EU: 'EU', GDPR: 'EU',
      TR: 'TR', TURKEY: 'TR', KVKK: 'TR',
    };
    const region = regionMap[regionStr] || 'INTL';
    const cmdId = `action-compliance-${region.toLowerCase()}`;
    if (ALL_COMMANDS.find((c) => c.id === cmdId)) {
      return { commandId: cmdId, confidence: 0.85 };
    }
  }

  return { confidence: 0 };
}

/**
 * Invalidate and rebuild the fuse index (call when commands change dynamically).
 */
export function rebuildIndex(): void {
  fuseInstance = null;
}
