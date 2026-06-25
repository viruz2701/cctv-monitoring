import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Badge, StatusBadge, HealthBadge, PriorityBadge, TicketStatusBadge, RoleBadge } from '../Badge';

// Mock i18next
vi.mock('react-i18next', () => ({
    useTranslation: () => ({
        t: (key: string) => key,
        i18n: { language: 'ru' },
    }),
}));

describe('Badge', () => {
    it('renders children', () => {
        render(<Badge>Active</Badge>);
        expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('applies variant classes', () => {
        const { rerender } = render(<Badge variant="success">Success</Badge>);
        expect(screen.getByText('Success').className).toContain('bg-emerald-50');

        rerender(<Badge variant="danger">Danger</Badge>);
        expect(screen.getByText('Danger').className).toContain('bg-red-50');

        rerender(<Badge variant="info">Info</Badge>);
        expect(screen.getByText('Info').className).toContain('bg-cyan-50');
    });

    it('renders dot indicator', () => {
        render(<Badge dot variant="success">Online</Badge>);
        const badge = screen.getByText('Online');
        expect(badge.className).toContain('inline-flex');
        // dot should have bg-emerald-500
        const dot = badge.querySelector('span');
        expect(dot?.className).toContain('bg-emerald-500');
    });

    it('sets aria-label when provided', () => {
        render(<Badge ariaLabel="Custom label">Content</Badge>);
        expect(screen.getByLabelText('Custom label')).toBeInTheDocument();
    });

    it('generates aria-label for dot variant', () => {
        render(<Badge dot variant="danger">Offline</Badge>);
        expect(screen.getByLabelText('Danger status')).toBeInTheDocument();
    });

    it('applies size classes', () => {
        const { rerender } = render(<Badge size="sm">Small</Badge>);
        expect(screen.getByText('Small').className).toContain('px-2');

        rerender(<Badge size="lg">Large</Badge>);
        expect(screen.getByText('Large').className).toContain('px-3');
    });
});

describe('StatusBadge', () => {
    it('renders online status', () => {
        render(<StatusBadge status="online" />);
        expect(screen.getByText('online')).toBeInTheDocument();
    });

    it('renders offline status', () => {
        render(<StatusBadge status="offline" />);
        expect(screen.getByText('offline')).toBeInTheDocument();
    });
});

describe('HealthBadge', () => {
    it('renders healthy status', () => {
        render(<HealthBadge health="healthy" />);
        expect(screen.getByText('healthy')).toBeInTheDocument();
    });
});

describe('PriorityBadge', () => {
    it('renders critical priority', () => {
        render(<PriorityBadge priority="critical" />);
        expect(screen.getByText('critical')).toBeInTheDocument();
    });
});

describe('TicketStatusBadge', () => {
    it('renders open status', () => {
        render(<TicketStatusBadge status="open" />);
        expect(screen.getByText('open')).toBeInTheDocument();
    });
});

describe('RoleBadge', () => {
    it('renders admin role', () => {
        render(<RoleBadge role="admin" />);
        expect(screen.getByText('admin')).toBeInTheDocument();
    });
});
