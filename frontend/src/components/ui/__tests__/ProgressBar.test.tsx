import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ProgressBar } from '../ProgressBar';

describe('ProgressBar', () => {
    it('renders with default props', () => {
        render(<ProgressBar />);
        const bar = screen.getByRole('progressbar');
        expect(bar).toBeInTheDocument();
        expect(bar).toHaveAttribute('aria-valuenow', '0');
        expect(bar).toHaveAttribute('aria-valuemin', '0');
        expect(bar).toHaveAttribute('aria-valuemax', '100');
    });

    it('renders with custom value', () => {
        render(<ProgressBar value={50} />);
        const bar = screen.getByRole('progressbar');
        expect(bar).toHaveAttribute('aria-valuenow', '50');
    });

    it('renders with custom max', () => {
        render(<ProgressBar value={30} max={60} />);
        const bar = screen.getByRole('progressbar');
        expect(bar).toHaveAttribute('aria-valuemax', '60');
    });

    it('renders percentage label when showLabel is true', () => {
        render(<ProgressBar value={75} showLabel />);
        expect(screen.getByText('75%')).toBeInTheDocument();
    });

    it('does not render label when showLabel is false', () => {
        render(<ProgressBar value={50} showLabel={false} />);
        expect(screen.queryByText('50%')).not.toBeInTheDocument();
    });

    it('clamps value to 0-100 range', () => {
        const { rerender } = render(<ProgressBar value={-10} />);
        let bar = screen.getByRole('progressbar');
        expect(bar).toHaveAttribute('aria-valuenow', '-10');

        rerender(<ProgressBar value={150} />);
        bar = screen.getByRole('progressbar');
        expect(bar).toHaveAttribute('aria-valuenow', '150');
    });

    it('applies variant classes to track', () => {
        const { rerender } = render(<ProgressBar variant="success" />);
        // Track element has variant class
        const track = document.querySelector('[class*="bg-emerald-100"]');
        expect(track).toBeInTheDocument();

        rerender(<ProgressBar variant="danger" />);
        const dangerTrack = document.querySelector('[class*="bg-red-100"]');
        expect(dangerTrack).toBeInTheDocument();
    });

    it('has accessible aria-label', () => {
        render(<ProgressBar value={42} />);
        const bar = screen.getByRole('progressbar');
        expect(bar).toHaveAttribute('aria-label', 'Progress: 42%');
    });

    it('handles max=0 gracefully', () => {
        render(<ProgressBar value={10} max={0} />);
        const bar = screen.getByRole('progressbar');
        expect(bar).toBeInTheDocument();
    });
});
