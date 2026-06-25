import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Tabs } from '../Tabs';

const mockTabs = [
    { id: 'tab1', label: 'Tab One' },
    { id: 'tab2', label: 'Tab Two', badge: 5 },
    { id: 'tab3', label: 'Tab Three', disabled: true },
];

describe('Tabs', () => {
    it('renders all tab labels', () => {
        render(
            <Tabs tabs={mockTabs} onChange={vi.fn()}>
                <div>Content</div>
            </Tabs>
        );
        expect(screen.getByText('Tab One')).toBeInTheDocument();
        expect(screen.getByText('Tab Two')).toBeInTheDocument();
        expect(screen.getByText('Tab Three')).toBeInTheDocument();
    });

    it('renders children content', () => {
        render(
            <Tabs tabs={mockTabs} onChange={vi.fn()}>
                <div>Tab Content</div>
            </Tabs>
        );
        expect(screen.getByText('Tab Content')).toBeInTheDocument();
    });

    it('calls onChange when a tab is clicked', async () => {
        const handleChange = vi.fn();
        const user = userEvent.setup();
        render(
            <Tabs tabs={mockTabs} onChange={handleChange}>
                <div>Content</div>
            </Tabs>
        );

        await user.click(screen.getByText('Tab Two'));
        expect(handleChange).toHaveBeenCalledWith('tab2');
    });

    it('does not call onChange for disabled tab', async () => {
        const handleChange = vi.fn();
        const user = userEvent.setup();
        render(
            <Tabs tabs={mockTabs} onChange={handleChange}>
                <div>Content</div>
            </Tabs>
        );

        await user.click(screen.getByText('Tab Three'));
        expect(handleChange).not.toHaveBeenCalled();
    });

    it('shows active tab with correct styling', () => {
        render(
            <Tabs tabs={mockTabs} activeTab="tab1" onChange={vi.fn()}>
                <div>Content</div>
            </Tabs>
        );

        // Active tab has blue border
        const tabs = screen.getAllByRole('button');
        expect(tabs[0].className).toContain('border-blue-600');
        // Inactive tab does not
        expect(tabs[1].className).toContain('border-transparent');
    });

    it('renders badge count', () => {
        render(
            <Tabs tabs={mockTabs} onChange={vi.fn()}>
                <div>Content</div>
            </Tabs>
        );
        expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('applies pills variant styles', () => {
        render(
            <Tabs tabs={mockTabs} onChange={vi.fn()} variant="pills">
                <div>Content</div>
            </Tabs>
        );
        const tabs = screen.getAllByRole('button');
        // Pills variant has rounded-lg
        expect(tabs[0].className).toContain('rounded-lg');
        // Active tab has blue bg
        expect(tabs[0].className).toContain('bg-blue-600');
    });

    it('applies underline variant styles', () => {
        render(
            <Tabs tabs={mockTabs} activeTab="tab2" onChange={vi.fn()} variant="underline">
                <div>Content</div>
            </Tabs>
        );
        const tabs = screen.getAllByRole('button');
        // Inactive tab should have muted text
        expect(tabs[0].className).toContain('text-slate-500');
        // Active tab should have blue text
        expect(tabs[1].className).toContain('text-blue-600');
    });
});
