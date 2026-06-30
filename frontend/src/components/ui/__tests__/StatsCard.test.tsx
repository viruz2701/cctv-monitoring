import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StatsCard, MiniStatsCard } from '../StatsCard';
import { Camera, TrendingUp } from '../Icons';

describe('StatsCard', () => {
    it('renders title and value', () => {
        render(
            <StatsCard title="Total Cameras" value="42" icon={Camera} />
        );
        expect(screen.getByText('Total Cameras')).toBeInTheDocument();
        expect(screen.getByText('42')).toBeInTheDocument();
    });

    it('renders icon with default colors', () => {
        const { container } = render(
            <StatsCard title="Cameras" value="24" icon={Camera} />
        );
        const iconContainer = container.querySelector('.p-3');
        expect(iconContainer?.className).toContain('bg-blue-50');
        const icon = container.querySelector('svg');
        // SVG className in JSDOM returns SVGAnimatedString, check via getAttribute
        const iconClass = icon?.getAttribute('class') || '';
        expect(iconClass).toContain('text-blue-600');
    });

    it('renders custom icon colors', () => {
        const { container } = render(
            <StatsCard
                title="Alerts"
                value="7"
                icon={Camera}
                iconColor="text-red-600"
                iconBgColor="bg-red-50"
            />
        );
        const iconContainer = container.querySelector('.p-3');
        expect(iconContainer?.className).toContain('bg-red-50');
        const icon = container.querySelector('svg');
        const iconClass = icon?.getAttribute('class') || '';
        expect(iconClass).toContain('text-red-600');
    });

    it('renders subtitle when provided', () => {
        render(
            <StatsCard
                title="Cameras"
                value="24"
                icon={Camera}
                subtitle="Last updated: 2 min ago"
            />
        );
        expect(screen.getByText('Last updated: 2 min ago')).toBeInTheDocument();
    });

    it('does not render subtitle when not provided', () => {
        render(
            <StatsCard title="Cameras" value="24" icon={Camera} />
        );
        expect(screen.queryByText('Last updated: 2 min ago')).not.toBeInTheDocument();
    });

    it('renders trend with up direction', () => {
        render(
            <StatsCard
                title="Cameras"
                value="24"
                icon={Camera}
                trend={{ value: 12, label: 'vs last week', direction: 'up' }}
            />
        );
        expect(screen.getByText('12%')).toBeInTheDocument();
        expect(screen.getByText('vs last week')).toBeInTheDocument();
        // Up trend should have emerald/positive color
        const trendValue = screen.getByText('12%');
        expect(trendValue.className).toContain('text-emerald-600');
    });

    it('renders trend with down direction', () => {
        render(
            <StatsCard
                title="Offline Devices"
                value="3"
                icon={Camera}
                trend={{ value: 5, label: 'vs last week', direction: 'down' }}
            />
        );
        expect(screen.getByText('5%')).toBeInTheDocument();
        const trendValue = screen.getByText('5%');
        expect(trendValue.className).toContain('text-red-600');
    });

    it('does not render trend when not provided', () => {
        render(
            <StatsCard title="Cameras" value="24" icon={Camera} />
        );
        expect(screen.queryByText('%')).not.toBeInTheDocument();
    });

    it('applies custom className', () => {
        const { container } = render(
            <StatsCard title="Cameras" value="24" icon={Camera} className="my-custom-class" />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('my-custom-class');
    });

    it('renders with zero value', () => {
        render(
            <StatsCard title="Active Alerts" value={0} icon={Camera} />
        );
        expect(screen.getByText('0')).toBeInTheDocument();
    });

    it('renders with empty string value', () => {
        render(
            <StatsCard title="Uptime" value="—" icon={Camera} />
        );
        expect(screen.getByText('—')).toBeInTheDocument();
    });

    it('renders numeric value correctly', () => {
        render(
            <StatsCard title="Resolution Time" value={99.5} icon={Camera} />
        );
        expect(screen.getByText('99.5')).toBeInTheDocument();
    });
});

describe('MiniStatsCard', () => {
    it('renders title and value', () => {
        render(
            <MiniStatsCard title="Cameras" value="24" icon={Camera} />
        );
        expect(screen.getByText('24')).toBeInTheDocument();
        expect(screen.getByText('Cameras')).toBeInTheDocument();
    });

    it('renders with default blue color', () => {
        const { container } = render(
            <MiniStatsCard title="Cameras" value="24" icon={Camera} />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('border-l-blue-500');
        const iconContainer = container.querySelector('.p-2');
        expect(iconContainer?.className).toContain('bg-blue-50');
    });

    it('renders with custom color', () => {
        const { container } = render(
            <MiniStatsCard title="Alerts" value="7" icon={Camera} color="red" />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('border-l-red-500');
        const iconContainer = container.querySelector('.p-2');
        expect(iconContainer?.className).toContain('bg-red-50');
    });

    it('renders with green color', () => {
        const { container } = render(
            <MiniStatsCard title="Online" value="22" icon={Camera} color="green" />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('border-l-emerald-500');
    });

    it('renders with amber color', () => {
        const { container } = render(
            <MiniStatsCard title="Warnings" value="3" icon={Camera} color="amber" />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('border-l-amber-500');
    });

    it('renders with purple color', () => {
        const { container } = render(
            <MiniStatsCard title="Users" value="18" icon={Camera} color="purple" />
        );
        const card = container.firstChild as HTMLElement;
        expect(card.className).toContain('border-l-purple-500');
    });
});
