import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { EmptyState } from '../EmptyState';
import { HardDrive } from './Icons';

describe('EmptyState', () => {
    it('renders with icon, title and description', () => {
        render(
            <EmptyState
                icon={<HardDrive data-testid="icon" />}
                title="No devices"
                description="Add your first device"
            />
        );
        expect(screen.getByText('No devices')).toBeInTheDocument();
        expect(screen.getByText('Add your first device')).toBeInTheDocument();
    });

    it('renders hint text when provided', () => {
        render(
            <EmptyState
                icon={<HardDrive />}
                title="Empty"
                hint="Try adding something"
            />
        );
        expect(screen.getByText('Try adding something')).toBeInTheDocument();
    });

    it('renders primary action button', () => {
        const handleAction = vi.fn();
        render(
            <EmptyState
                icon={<HardDrive />}
                title="No items"
                action={{ label: 'Add Item', onClick: handleAction }}
            />
        );
        expect(screen.getByText('Add Item')).toBeInTheDocument();
    });

    it('calls action onClick when clicked', async () => {
        const handleAction = vi.fn();
        const user = userEvent.setup();
        render(
            <EmptyState
                icon={<HardDrive />}
                title="No items"
                action={{ label: 'Add Item', onClick: handleAction }}
            />
        );
        await user.click(screen.getByText('Add Item'));
        expect(handleAction).toHaveBeenCalledTimes(1);
    });

    it('renders secondary action button', () => {
        render(
            <EmptyState
                icon={<HardDrive />}
                title="No items"
                secondaryAction={{ label: 'Learn More', onClick: vi.fn() }}
            />
        );
        expect(screen.getByText('Learn More')).toBeInTheDocument();
    });

    it('applies size variants', () => {
        const { rerender } = render(
            <EmptyState icon={<HardDrive />} title="Test" size="sm" />
        );
        expect(screen.getByText('Test').className).toContain('text-sm');

        rerender(
            <EmptyState icon={<HardDrive />} title="Test" size="lg" />
        );
        expect(screen.getByText('Test').className).toContain('text-xl');
    });

    it('renders without description or hint', () => {
        render(
            <EmptyState icon={<HardDrive />} title="Minimal" />
        );
        expect(screen.getByText('Minimal')).toBeInTheDocument();
        expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });
});
