import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Button, IconButton } from '../Button';
import { Settings } from 'lucide-react';

describe('Button', () => {
    it('renders children', () => {
        render(<Button>Click me</Button>);
        expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument();
    });

    it('calls onClick handler when clicked', async () => {
        const handleClick = vi.fn();
        const user = userEvent.setup();
        render(<Button onClick={handleClick}>Click</Button>);
        await user.click(screen.getByRole('button'));
        expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('is disabled when disabled prop is true', () => {
        render(<Button disabled>Disabled</Button>);
        expect(screen.getByRole('button')).toBeDisabled();
    });

    it('shows loading state and disables button', () => {
        render(<Button loading>Loading</Button>);
        const btn = screen.getByRole('button');
        expect(btn).toBeDisabled();
        expect(btn).toHaveAttribute('aria-busy', 'true');
    });

    it('applies variant classes correctly', () => {
        const { rerender } = render(<Button variant="primary">Primary</Button>);
        expect(screen.getByRole('button').className).toContain('bg-blue-600');

        rerender(<Button variant="danger">Danger</Button>);
        expect(screen.getByRole('button').className).toContain('bg-red-600');

        rerender(<Button variant="ghost">Ghost</Button>);
        expect(screen.getByRole('button').className).toContain('text-slate-700');
    });

    it('applies size classes correctly', () => {
        const { rerender } = render(<Button size="sm">Small</Button>);
        expect(screen.getByRole('button').className).toContain('px-3');

        rerender(<Button size="lg">Large</Button>);
        expect(screen.getByRole('button').className).toContain('px-5');
    });

    it('renders icon on the left by default', () => {
        render(<Button icon={<Settings data-testid="icon" />}>With Icon</Button>);
        const btn = screen.getByRole('button');
        expect(btn.innerHTML).toContain('lucide');
    });

    it('applies fullWidth class', () => {
        render(<Button fullWidth>Full Width</Button>);
        expect(screen.getByRole('button').className).toContain('w-full');
    });

    it('does not render icon when loading', () => {
        render(<Button loading icon={<Settings data-testid="icon" />}>Loading</Button>);
        // Loader2 spinner should be present instead of the icon
        expect(screen.getByRole('button').querySelector('.animate-spin')).toBeInTheDocument();
    });
});

describe('IconButton', () => {
    it('renders with aria-label', () => {
        render(<IconButton icon={<Settings />} label="Settings" />);
        expect(screen.getByLabelText('Settings')).toBeInTheDocument();
    });

    it('applies variant classes', () => {
        render(<IconButton icon={<Settings />} label="Settings" variant="primary" />);
        expect(screen.getByRole('button').className).toContain('bg-blue-600');
    });
});
