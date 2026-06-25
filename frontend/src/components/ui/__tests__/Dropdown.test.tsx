import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Dropdown, type DropdownItem } from '../Dropdown';

const mockItems: DropdownItem[] = [
    { id: 'edit', label: 'Edit' },
    { id: 'delete', label: 'Delete', danger: true },
    { id: 'separator', divider: true, label: '' },
    { id: 'export', label: 'Export', disabled: true },
];

describe('Dropdown', () => {
    it('renders anchor element', () => {
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} />
        );
        expect(screen.getByText('Menu')).toBeInTheDocument();
    });

    it('shows menu items when clicked', async () => {
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} />
        );

        await user.click(screen.getByText('Menu'));
        expect(screen.getByRole('menu')).toBeInTheDocument();
        expect(screen.getByText('Edit')).toBeInTheDocument();
    });

    it('calls onSelect when an item is clicked', async () => {
        const handleSelect = vi.fn();
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={handleSelect} anchor={<span>Menu</span>} />
        );

        await user.click(screen.getByText('Menu'));
        await user.click(screen.getByText('Edit'));
        expect(handleSelect).toHaveBeenCalledWith(mockItems[0]);
    });

    it('closes menu after selection', async () => {
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} />
        );

        await user.click(screen.getByText('Menu'));
        await user.click(screen.getByText('Edit'));
        expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('does not open when disabled', async () => {
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} disabled />
        );

        await user.click(screen.getByText('Menu'));
        expect(screen.queryByRole('menu')).not.toBeInTheDocument();
    });

    it('disables clicking disabled items', async () => {
        const handleSelect = vi.fn();
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={handleSelect} anchor={<span>Menu</span>} />
        );

        await user.click(screen.getByText('Menu'));
        const disabledBtn = screen.getByText('Export').closest('button');
        expect(disabledBtn).toBeDisabled();
    });

    it('renders divider as separator', async () => {
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} />
        );

        await user.click(screen.getByText('Menu'));
        const separators = document.querySelectorAll('[role="separator"]');
        expect(separators.length).toBe(1);
    });

    it('has aria-haspopup and aria-expanded attributes', async () => {
        const user = userEvent.setup();
        render(
            <Dropdown items={mockItems} onSelect={vi.fn()} anchor={<span>Menu</span>} />
        );

        const trigger = screen.getByText('Menu').closest('[role="button"]')!;
        expect(trigger).toHaveAttribute('aria-haspopup', 'true');
        expect(trigger).toHaveAttribute('aria-expanded', 'false');

        await user.click(screen.getByText('Menu'));
        expect(trigger).toHaveAttribute('aria-expanded', 'true');
    });
});
