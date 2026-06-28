import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, it, expect, vi } from 'vitest';
import { Breadcrumbs, type BreadcrumbItem } from '../Breadcrumbs';

// Mock react-i18next
vi.mock('react-i18next', () => ({
    useTranslation: () => ({
        t: (key: string) => {
            const translations: Record<string, string> = {
                'breadcrumb': 'Breadcrumb',
                'back': 'Back',
                'Dashboard': 'Dashboard',
                'Devices': 'Devices',
                'NVR-01': 'NVR-01',
            };
            return translations[key] || key;
        },
    }),
}));

const mockItems: BreadcrumbItem[] = [
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Devices', href: '/devices' },
    { label: 'NVR-01' },
];

const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <BrowserRouter>{children}</BrowserRouter>
);

describe('Breadcrumbs', () => {
    it('renders all items', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        // Text appears in both desktop and mobile views, use getAllByText
        expect(screen.getAllByText('Dashboard').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('Devices').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('NVR-01').length).toBeGreaterThanOrEqual(1);
    });

    it('renders last item without href as plain text', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        const items = screen.getAllByRole('listitem');
        const lastItem = items[items.length - 1];
        // Last item should not contain a link
        expect(lastItem.querySelector('a')).toBeNull();
    });

    it('sets aria-current="page" on last item', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        const lastItems = screen.getAllByText('NVR-01');
        // At least one occurrence should have aria-current
        const hasAriaCurrent = lastItems.some(el => el.closest('[aria-current="page"]'));
        expect(hasAriaCurrent).toBe(true);
    });

    it('renders links for non-last items with href', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        const dashboardLinks = screen.getAllByText('Dashboard');
        const dashboardLink = dashboardLinks[0].closest('a');
        expect(dashboardLink).toHaveAttribute('href', '/dashboard');

        const devicesLinks = screen.getAllByText('Devices');
        const devicesLink = devicesLinks[0].closest('a');
        expect(devicesLink).toHaveAttribute('href', '/devices');
    });

    it('renders separator between items', () => {
        const { container } = render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        // ChevronRight icons are used as separators
        const separators = container.querySelectorAll('svg.lucide-chevron-right');
        // 2 separators for 3 items
        expect(separators.length).toBeGreaterThanOrEqual(2);
    });

    it('renders custom separator when provided', () => {
        const { container } = render(
            <Breadcrumbs items={mockItems} separator={<span data-testid="custom-sep">/</span>} />,
            { wrapper: Wrapper }
        );
        expect(container.querySelector('[data-testid="custom-sep"]')).toBeInTheDocument();
    });

    it('collapses items when maxItems is exceeded', () => {
        const manyItems: BreadcrumbItem[] = [
            { label: 'Home', href: '/' },
            { label: 'Organization', href: '/org' },
            { label: 'Region', href: '/region' },
            { label: 'Site', href: '/site' },
            { label: 'Building', href: '/building' },
            { label: 'Floor', href: '/floor' },
            { label: 'Room', href: '/room' },
            { label: 'NVR-01' },
        ];

        render(<Breadcrumbs items={manyItems} maxItems={4} />, { wrapper: Wrapper });

        // Text appears in both desktop and mobile views
        expect(screen.getAllByText('Home').length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText('…')).toBeInTheDocument();
        expect(screen.getAllByText('NVR-01').length).toBeGreaterThanOrEqual(1);
    });

    it('does not collapse when items are within maxItems', () => {
        render(<Breadcrumbs items={mockItems} maxItems={5} />, { wrapper: Wrapper });
        expect(screen.queryByText('…')).not.toBeInTheDocument();
        expect(screen.getAllByText('Dashboard').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('Devices').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('NVR-01').length).toBeGreaterThanOrEqual(1);
    });

    it('applies custom className', () => {
        const { container } = render(
            <Breadcrumbs items={mockItems} className="my-custom-class" />,
            { wrapper: Wrapper }
        );
        const nav = container.querySelector('nav');
        expect(nav?.className).toContain('my-custom-class');
    });

    it('has accessible aria-label on nav', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        expect(screen.getByLabelText('Breadcrumb')).toBeInTheDocument();
    });

    it('shows mobile view with back link when prev item has href', () => {
        render(<Breadcrumbs items={mockItems} />, { wrapper: Wrapper });
        // Mobile: back button should have aria-label="Back"
        const backBtn = screen.getByLabelText('Back');
        expect(backBtn).toBeInTheDocument();
    });

    it('renders single item without separators', () => {
        const singleItem: BreadcrumbItem[] = [{ label: 'Dashboard' }];
        const { container } = render(<Breadcrumbs items={singleItem} />, { wrapper: Wrapper });
        // No separators for single item
        const separators = container.querySelectorAll('svg.lucide-chevron-right');
        expect(separators.length).toBe(0);
    });
});
