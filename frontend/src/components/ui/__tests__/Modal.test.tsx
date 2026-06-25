import React from 'react';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { Modal, ConfirmModal } from '../Modal';

// Mock useFocusTrap and useTabIndex hooks
vi.mock('../../../hooks/useAccessibility', () => ({
    useFocusTrap: () => ({
        containerRef: { current: document.createElement('div') },
        handleKeyDown: vi.fn(),
    }),
    useTabIndex: vi.fn(),
}));

afterEach(() => {
    cleanup();
    // Clean up any leftover portal elements
    document.body.querySelectorAll('[role="dialog"]').forEach(el => el.remove());
});

describe('Modal', () => {
    it('renders nothing when closed', () => {
        render(
            <Modal isOpen={false} onClose={vi.fn()}>
                <p>Content</p>
            </Modal>
        );
        expect(screen.queryByText('Content')).not.toBeInTheDocument();
    });

    it('renders content when open', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()}>
                <p>Modal Content</p>
            </Modal>
        );
        // Content is rendered in portal (document.body)
        const modalRoot = document.body;
        expect(modalRoot.textContent).toContain('Modal Content');
    });

    it('renders title when provided', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()} title="Test Title">
                <p>Content</p>
            </Modal>
        );
        expect(document.body.textContent).toContain('Test Title');
    });

    it('calls onClose when close button is clicked', async () => {
        const handleClose = vi.fn();
        const user = userEvent.setup();
        render(
            <Modal isOpen={true} onClose={handleClose}>
                <p>Content</p>
            </Modal>
        );
        const closeBtn = document.querySelector('button[aria-label="Close modal"]');
        expect(closeBtn).toBeTruthy();
        await user.click(closeBtn!);
        expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('calls onClose when escape key is pressed', async () => {
        const handleClose = vi.fn();
        const user = userEvent.setup();
        render(
            <Modal isOpen={true} onClose={handleClose}>
                <p>Content</p>
            </Modal>
        );
        await user.keyboard('{Escape}');
        expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('calls onClose when backdrop is clicked', async () => {
        const handleClose = vi.fn();
        const user = userEvent.setup();
        render(
            <Modal isOpen={true} onClose={handleClose}>
                <p>Content</p>
            </Modal>
        );
        // Click the backdrop overlay
        const overlay = document.querySelector('.fixed.inset-0');
        expect(overlay).toBeTruthy();
        await user.click(overlay!);
        expect(handleClose).toHaveBeenCalledTimes(1);
    });

    it('renders footer when provided', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()} footer={<button>Save</button>}>
                <p>Content</p>
            </Modal>
        );
        expect(document.body.textContent).toContain('Save');
    });

    it('does not show close button when showClose is false', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()} showClose={false}>
                <p>Content</p>
            </Modal>
        );
        expect(document.querySelector('button[aria-label="Close modal"]')).not.toBeInTheDocument();
    });

    it('sets aria-modal and role="dialog"', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()} title="Aria Test">
                <p>Content</p>
            </Modal>
        );
        const dialog = document.querySelector('[role="dialog"]');
        expect(dialog).toBeInTheDocument();
        expect(dialog).toHaveAttribute('aria-modal', 'true');
    });

    it('prevents body scroll when open', () => {
        render(
            <Modal isOpen={true} onClose={vi.fn()}>
                <p>Content</p>
            </Modal>
        );
        expect(document.body.style.overflow).toBe('hidden');
    });

    it('restores body scroll when closed', () => {
        const { rerender } = render(
            <Modal isOpen={true} onClose={vi.fn()}>
                <p>Content</p>
            </Modal>
        );
        rerender(
            <Modal isOpen={false} onClose={vi.fn()}>
                <p>Content</p>
            </Modal>
        );
        expect(document.body.style.overflow).toBe('');
    });
});

describe('ConfirmModal', () => {
    it('renders title and message', () => {
        render(
            <ConfirmModal
                isOpen={true}
                onClose={vi.fn()}
                onConfirm={vi.fn()}
                title="Delete?"
                message="Are you sure?"
            />
        );
        expect(document.body.textContent).toContain('Delete?');
        expect(document.body.textContent).toContain('Are you sure?');
    });

    it('calls onConfirm when confirm button is clicked', async () => {
        const handleConfirm = vi.fn();
        const user = userEvent.setup();
        render(
            <ConfirmModal
                isOpen={true}
                onClose={vi.fn()}
                onConfirm={handleConfirm}
                title="Confirm Deletion"
                message="Proceed?"
                confirmText="Delete"
            />
        );
        await user.click(screen.getByText('Delete'));
        expect(handleConfirm).toHaveBeenCalledTimes(1);
    });

    it('calls onClose when cancel is clicked', async () => {
        const handleClose = vi.fn();
        const user = userEvent.setup();
        render(
            <ConfirmModal
                isOpen={true}
                onClose={handleClose}
                onConfirm={vi.fn()}
                title="Confirm"
                message="Proceed?"
            />
        );
        await user.click(screen.getByText('Cancel'));
        expect(handleClose).toHaveBeenCalledTimes(1);
    });
});
