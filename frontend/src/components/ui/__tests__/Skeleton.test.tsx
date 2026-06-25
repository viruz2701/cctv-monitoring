import React from 'react';
import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Skeleton, SkeletonLine, SkeletonAvatar } from '../Skeleton';

describe('Skeleton', () => {
    it('renders with default variant', () => {
        const { container } = render(<Skeleton />);
        const div = container.firstChild as HTMLElement;
        expect(div.className).toContain('rounded-lg');
        expect(div).toHaveAttribute('aria-hidden', 'true');
    });

    it('renders text variant', () => {
        const { container } = render(<Skeleton variant="text" />);
        const div = container.firstChild as HTMLElement;
        expect(div.className).toContain('rounded');
    });

    it('renders circular variant', () => {
        const { container } = render(<Skeleton variant="circular" />);
        const div = container.firstChild as HTMLElement;
        expect(div.className).toContain('rounded-full');
    });

    it('renders rectangular variant', () => {
        const { container } = render(<Skeleton variant="rectangular" />);
        const div = container.firstChild as HTMLElement;
        expect(div.className).toContain('rounded-none');
    });

    it('applies custom width and height', () => {
        const { container } = render(<Skeleton width={100} height={20} />);
        const div = container.firstChild as HTMLElement;
        expect(div.style.width).toBe('100px');
        expect(div.style.height).toBe('20px');
    });

    it('renders multiple elements with count', () => {
        const { container } = render(<Skeleton count={3} />);
        expect(container.children.length).toBe(3);
    });
});

describe('SkeletonLine', () => {
    it('renders two lines per block', () => {
        const { container } = render(<SkeletonLine />);
        const wrapper = container.firstChild as HTMLElement;
        expect(wrapper.children.length).toBe(2);
    });

    it('renders multiple blocks with count', () => {
        const { container } = render(<SkeletonLine count={2} />);
        expect(container.children.length).toBe(2);
    });
});

describe('SkeletonAvatar', () => {
    it('renders with default size', () => {
        const { container } = render(<SkeletonAvatar />);
        const wrapper = container.firstChild as HTMLElement;
        expect(wrapper).toBeInTheDocument();
        expect(wrapper.className).toContain('flex');
        // The actual avatar is a child div inside the wrapper
        const avatar = wrapper.firstChild as HTMLElement;
        expect(avatar.className).toContain('rounded-full');
        expect(avatar.className).toContain('w-10');
    });

    it('renders with custom size', () => {
        const { container } = render(<SkeletonAvatar size="xl" />);
        const wrapper = container.firstChild as HTMLElement;
        const avatar = wrapper.firstChild as HTMLElement;
        expect(avatar).toBeInTheDocument();
        expect(avatar.className).toContain('rounded-full');
        expect(avatar.className).toContain('w-20');
    });
});
