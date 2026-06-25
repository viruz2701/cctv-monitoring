import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Tooltip } from '../Tooltip';

describe('Tooltip', () => {
    it('renders children', () => {
        render(
            <Tooltip content="Tooltip text">
                <button>Hover me</button>
            </Tooltip>
        );
        expect(screen.getByText('Hover me')).toBeInTheDocument();
    });

    it('shows tooltip on hover', async () => {
        const user = userEvent.setup();
        render(
            <Tooltip content="Tooltip text" delay={0}>
                <button>Hover me</button>
            </Tooltip>
        );

        await user.hover(screen.getByText('Hover me'));

        // Tooltip should be visible (opacity-100 instead of opacity-0)
        const tooltipEl = document.querySelector('[role="tooltip"]');
        expect(tooltipEl).toBeInTheDocument();
        expect(tooltipEl.className).toContain('opacity-100');
    });

    it('has role="tooltip" in DOM', () => {
        render(
            <Tooltip content="Tooltip content" delay={0}>
                <button>Hover</button>
            </Tooltip>
        );

        // Tooltip is always in DOM but hidden
        const tooltip = document.querySelector('[role="tooltip"]');
        expect(tooltip).toBeInTheDocument();
    });

    it('applies position classes to tooltip', () => {
        const { rerender } = render(
            <Tooltip content="Content" position="bottom">
                <button>Hover</button>
            </Tooltip>
        );

        let tooltip = document.querySelector('[role="tooltip"]');
        expect(tooltip.className).toContain('top-full');

        rerender(
            <Tooltip content="Content" position="left">
                <button>Hover</button>
            </Tooltip>
        );

        tooltip = document.querySelector('[role="tooltip"]');
        expect(tooltip.className).toContain('right-full');

        rerender(
            <Tooltip content="Content" position="right">
                <button>Hover</button>
            </Tooltip>
        );

        tooltip = document.querySelector('[role="tooltip"]');
        expect(tooltip.className).toContain('left-full');
    });

    it('hides tooltip on mouse leave', async () => {
        const user = userEvent.setup();
        render(
            <Tooltip content="Tooltip text" delay={0} hideDelay={0}>
                <button>Hover me</button>
            </Tooltip>
        );

        const trigger = screen.getByText('Hover me');
        const tooltipEl = document.querySelector('[role="tooltip"]');

        // Show
        await user.hover(trigger);
        expect(tooltipEl.className).toContain('opacity-100');

        // Hide
        await user.unhover(trigger);
        expect(tooltipEl.className).toContain('opacity-0');
    });
});
