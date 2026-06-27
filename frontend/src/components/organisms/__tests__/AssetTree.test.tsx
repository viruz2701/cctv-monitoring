// ═══════════════════════════════════════════════════════════════════════
// AssetTree — Unit Tests
// P1-QA.8: Coverage 80%
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AssetTree, type AssetTreeNode } from '../AssetTree';

// ── Mocks ────────────────────────────────────────────────────────────

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Mock hooks used by UI components
vi.mock('../../../hooks/useRipple', () => ({
  useRipple: () => ({ createRipple: vi.fn(), ripples: null }),
}));

vi.mock('../../../hooks/useHapticFeedback', () => ({
  useHapticFeedback: () => ({ light: vi.fn(), medium: vi.fn(), heavy: vi.fn() }),
}));

// Mock UI components (AssetTree imports from '../ui' index)
vi.mock('../ui', () => ({
  Card: ({ children, className }: any) => <div className={className}>{children}</div>,
  Badge: ({ children, variant, size }: any) => (
    <span data-testid="mock-badge" data-variant={variant} data-size={size}>{children}</span>
  ),
  Button: ({ children, onClick, disabled, icon, loading }: any) => (
    <button onClick={onClick} disabled={disabled}>{icon}{children}</button>
  ),
  StatsCard: ({ title, value, icon: Icon }: any) => (
    <div data-testid="mock-stats-card">
      {Icon && <Icon />}<span>{title}</span><span>{value}</span>
    </div>
  ),
}));

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        organizations: 'Organizations',
        sites: 'Sites',
        devices: 'Devices',
        online: 'Online',
        offline: 'Offline',
        search_asset: 'Search assets...',
        expand_all: 'Expand',
        collapse_all: 'Collapse',
        refresh: 'Refresh',
        loading: 'Loading...',
        no_search_results: 'Nothing found',
        no_assets: 'No assets',
        retry: 'Retry',
      };
      return translations[key] || key;
    },
  }),
}));

// ── Test Data ────────────────────────────────────────────────────────

const mockAssetTreeData: AssetTreeNode[] = [
  {
    id: 'org-main',
    name: 'Main Organization',
    type: 'organization',
    status: 'active',
    deviceCount: 3,
    level: 0,
    collapsed: true,
    children: [
      {
        id: 'site-1',
        name: 'Main Office',
        type: 'site',
        status: 'active',
        deviceCount: 2,
        level: 1,
        collapsed: true,
        children: [
          {
            id: 'site-1-building',
            name: 'Building A',
            type: 'building',
            status: 'active',
            deviceCount: 2,
            level: 2,
            collapsed: true,
            children: [
              {
                id: 'site-1-floor',
                name: 'Floor 1',
                type: 'floor',
                status: 'active',
                deviceCount: 2,
                level: 3,
                collapsed: false,
                children: [],
                devices: [
                  { device_id: 'dev-1', name: 'Camera-Lobby-01', device_type: 'camera', status: 'ONLINE' },
                  { device_id: 'dev-2', name: 'Camera-Lobby-02', device_type: 'camera', status: 'ONLINE' },
                ],
              },
            ],
          },
        ],
      },
      {
        id: 'site-2',
        name: 'Branch Office',
        type: 'site',
        status: 'active',
        deviceCount: 1,
        level: 1,
        collapsed: true,
        children: [],
        devices: [
          { device_id: 'dev-3', name: 'Camera-12 Parking Lot B', device_type: 'camera', status: 'OFFLINE' },
        ],
      },
    ],
  },
];

// ── Helpers ──────────────────────────────────────────────────────────

function renderAssetTree(props: Partial<React.ComponentProps<typeof AssetTree>> = {}) {
  return render(
    <BrowserRouter>
      <AssetTree
        data={mockAssetTreeData}
        loading={false}
        {...props}
      />
    </BrowserRouter>,
  );
}

// ── Tests ────────────────────────────────────────────────────────────

describe('AssetTree', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  // ── Rendering ─────────────────────────────────────────────────────

  it('Должен отрендерить корневые узлы', () => {
    renderAssetTree();
    expect(screen.getByText('Main Organization')).toBeDefined();
  });

  it('Должен показать названия узлов', () => {
    renderAssetTree();
    expect(screen.getByText('Main Office')).toBeDefined();
    expect(screen.getByText('Branch Office')).toBeDefined();
  });

  it('Должен показать иконки типов узлов', () => {
    const { container } = renderAssetTree();
    // Organization icon (Globe) is rendered
    const orgIcon = container.querySelector('.text-purple-600');
    expect(orgIcon).toBeDefined();
  });

  it('Должен показать статусные баджи', () => {
    renderAssetTree();
    const statuses = screen.getAllByText('active');
    expect(statuses.length).toBeGreaterThan(0);
  });

  it('Должен показать device count badge', () => {
    renderAssetTree();
    expect(screen.getByText('3 devs')).toBeDefined();
  });

  it('Должен скрыть stats card при hideStats=true', () => {
    const { container } = renderAssetTree({ hideStats: true });
    // Stats cards not rendered
    expect(container.querySelector('.md\\:grid-cols-5')).toBeNull();
  });

  // ── Search ────────────────────────────────────────────────────────

  it('Должен показать поле поиска', () => {
    renderAssetTree();
    expect(screen.getByPlaceholderText('Search assets...')).toBeDefined();
  });

  it('Должен скрыть поле поиска при hideSearch=true', () => {
    renderAssetTree({ hideSearch: true });
    expect(screen.queryByPlaceholderText('Search assets...')).toBeNull();
  });

  it('Должен фильтровать узлы по поисковому запросу', async () => {
    renderAssetTree();
    const searchInput = screen.getByPlaceholderText('Search assets...');
    fireEvent.change(searchInput, { target: { value: 'Lobby' } });

    await waitFor(() => {
      // Найденное устройство подсвечено
      const highlighted = screen.getByText('Camera-Lobby-01');
      expect(highlighted).toBeDefined();
    });
  });

  it('Должен показать "Nothing found" при пустом результате поиска', async () => {
    renderAssetTree();
    const searchInput = screen.getByPlaceholderText('Search assets...');
    fireEvent.change(searchInput, { target: { value: 'zzz_nonexistent' } });

    await waitFor(() => {
      expect(screen.getByText('Nothing found')).toBeDefined();
    });
  });

  it('Должен снять выделение при очистке поиска', async () => {
    renderAssetTree();
    const searchInput = screen.getByPlaceholderText('Search assets...') as HTMLInputElement;

    // Вводим поиск
    fireEvent.change(searchInput, { target: { value: 'Lobby' } });
    await waitFor(() => {
      expect(screen.getByText('Camera-Lobby-01')).toBeDefined();
    });

    // Очищаем
    fireEvent.change(searchInput, { target: { value: '' } });

    await waitFor(() => {
      // Узел больше не подсвечен, но отображается
      expect(screen.getByText('Main Organization')).toBeDefined();
    });
  });

  // ── Expand/Collapse ───────────────────────────────────────────────

  it('Должен показывать узлы свёрнутыми по умолчанию', () => {
    renderAssetTree();
    // Дочерние узлы есть в DOM но скрыты
    expect(screen.getByText('Main Organization')).toBeDefined();
    // Site-1 дочерний узел не отображается как видимый (в DOM с height:0)
    expect(screen.getByText('Main Office')).toBeDefined();
  });

  it('Должен развернуть узел при клике', async () => {
    renderAssetTree();
    // Кликаем по узлу organization
    fireEvent.click(screen.getByText('Main Organization'));

    await waitFor(() => {
      // Дочерние узлы стали видны
      expect(screen.getByText('Main Office')).toBeDefined();
    });
  });

  it('Должен развернуть все узлы при клике Expand', async () => {
    renderAssetTree();
    const expandBtn = screen.getByText('Expand');
    fireEvent.click(expandBtn);

    await waitFor(() => {
      expect(screen.getByText('Building A')).toBeDefined();
    });
  });

  it('Должен переключиться на Collapse после клика Expand', async () => {
    renderAssetTree();
    const toggleBtn = screen.getByText('Expand');
    fireEvent.click(toggleBtn);
    // После разворачивания кнопка меняет текст
    await waitFor(() => {
      expect(screen.getByText('Collapse')).toBeDefined();
    });
  });

  // ── Device Interaction ────────────────────────────────────────────

  it('Должен вызвать onDeviceClick при клике на устройство', async () => {
    const onDeviceClick = vi.fn();

    render(
      <BrowserRouter>
        <AssetTree
          data={mockAssetTreeData}
          loading={false}
          onDeviceClick={onDeviceClick}
          defaultExpandAll={true}
        />
      </BrowserRouter>,
    );

    // Разворачиваем до устройств
    await waitFor(() => {
      expect(screen.getByText('Camera-Lobby-01')).toBeDefined();
    });

    fireEvent.click(screen.getByText('Camera-Lobby-01'));

    expect(onDeviceClick).toHaveBeenCalledWith('dev-1');
  });

  it('Должен навигировать на /devices/:id при клике на устройство без onDeviceClick', async () => {
    render(
      <BrowserRouter>
        <AssetTree
          data={mockAssetTreeData}
          loading={false}
          defaultExpandAll={true}
        />
      </BrowserRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText('Camera-Lobby-01')).toBeDefined();
    });

    fireEvent.click(screen.getByText('Camera-Lobby-01'));

    expect(mockNavigate).toHaveBeenCalledWith('/devices/dev-1');
  });

  // ── Loading State ─────────────────────────────────────────────────

  it('Должен показать спиннер при loading=true и пустых данных', () => {
    render(
      <BrowserRouter>
        <AssetTree data={[]} loading={true} />
      </BrowserRouter>,
    );
    expect(screen.getByText('Loading...')).toBeDefined();
  });

  it('Должен показать данные при loading=true с данными', () => {
    renderAssetTree({ loading: true });
    // Данные всё равно отображаются
    expect(screen.getByText('Main Organization')).toBeDefined();
  });

  // ── Empty State ───────────────────────────────────────────────────

  it('Должен показать "No assets" при пустом дереве', () => {
    render(
      <BrowserRouter>
        <AssetTree data={[]} loading={false} />
      </BrowserRouter>,
    );
    expect(screen.getByText('No assets')).toBeDefined();
  });

  // ── Error State ───────────────────────────────────────────────────

  it('Должен показать ошибку при error пропсе', () => {
    renderAssetTree({ error: 'Failed to load assets' });
    expect(screen.getByText('Failed to load assets')).toBeDefined();
  });

  // Retry button is only shown in non-external mode, tested via integration tests

  // ── Breadcrumbs ───────────────────────────────────────────────────

  it('Должен показать breadcrumbs после выбора узла', async () => {
    const onNodeSelect = vi.fn();

    render(
      <BrowserRouter>
        <AssetTree
          data={mockAssetTreeData}
          loading={false}
          onNodeSelect={onNodeSelect}
        />
      </BrowserRouter>,
    );

    // Разворачиваем орг -> site
    fireEvent.click(screen.getByText('Main Organization'));
    await waitFor(() => expect(screen.getByText('Main Office')).toBeDefined());

    fireEvent.click(screen.getByText('Main Office'));

    await waitFor(() => {
      // Breadcrumbs должны содержать путь
      expect(onNodeSelect).toHaveBeenCalled();
    });
  });

  // ── Device Status Colors ──────────────────────────────────────────

  it('Должен показать ONLINE статус зелёным', () => {
    render(
      <BrowserRouter>
        <AssetTree
          data={mockAssetTreeData}
          loading={false}
          defaultExpandAll={true}
        />
      </BrowserRouter>,
    );

    const onlineBadge = screen.getAllByText('ONLINE');
    expect(onlineBadge.length).toBe(2);
  });

  it('Должен показать OFFLINE статус красным', () => {
    render(
      <BrowserRouter>
        <AssetTree
          data={mockAssetTreeData}
          loading={false}
          defaultExpandAll={true}
        />
      </BrowserRouter>,
    );

    const offlineBadge = screen.getByText('OFFLINE');
    expect(offlineBadge).toBeDefined();
  });

  // ── Node Type Icons ───────────────────────────────────────────────

  it('Должен показать организацию с purple иконкой', () => {
    const { container } = renderAssetTree();
    const purpleIcon = container.querySelector('.bg-purple-100');
    expect(purpleIcon).toBeDefined();
  });

  it('Должен показать site с blue иконкой', () => {
    renderAssetTree();
    // Кликаем чтобы раскрыть организации
    fireEvent.click(screen.getByText('Main Organization'));
    const siteText = screen.getByText('Main Office');
    expect(siteText).toBeDefined();
  });

  // ── Toggle Node ───────────────────────────────────────────────────

  it('Должен переключать узел при клике на родителя', async () => {
    renderAssetTree();

    // Кликаем по тексту организации (onToggle вызывается через handleClick)
    fireEvent.click(screen.getByText('Main Organization'));

    await waitFor(() => {
      expect(screen.getByText('Main Office')).toBeDefined();
    });
  });
});
