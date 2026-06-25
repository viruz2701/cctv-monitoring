import React, { useState, useCallback, useRef } from 'react';
import { ConfirmModal } from '../components/ui';

interface ConfirmOptions {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    variant?: 'danger' | 'warning' | 'info';
}

interface ConfirmDialogHandle {
    confirm: (opts: ConfirmOptions) => Promise<boolean>;
    ConfirmDialog: React.ReactElement;
}

/**
 * UX-14.1.10: Promise-based confirmation dialog hook.
 *
 * Returns `confirm()` which resolves `true` when user confirms, `false` when cancelled.
 * The `ConfirmDialog` element should be rendered in the component tree via `{ConfirmDialog}`.
 *
 * @example
 * const { confirm, ConfirmDialog } = useConfirmAction();
 * if (await confirm({ title: 'Delete?', message: 'Are you sure?', variant: 'danger' })) {
 *   deleteItem(id);
 * }
 * // ...
 * return (<>{ConfirmDialog}<RestOfComponent /></>);
 */
export function useConfirmAction(): ConfirmDialogHandle {
    const [isOpen, setIsOpen] = useState(false);
    const [options, setOptions] = useState<ConfirmOptions>({
        title: '',
        message: '',
    });
    const resolveRef = useRef<((value: boolean) => void) | null>(null);

    const confirm = useCallback(async (opts: ConfirmOptions): Promise<boolean> => {
        return new Promise((resolve) => {
            resolveRef.current = resolve;
            setOptions(opts);
            setIsOpen(true);
        });
    }, []);

    const handleConfirm = useCallback(() => {
        resolveRef.current?.(true);
        resolveRef.current = null;
        setIsOpen(false);
    }, []);

    const handleClose = useCallback(() => {
        resolveRef.current?.(false);
        resolveRef.current = null;
        setIsOpen(false);
    }, []);

    const ConfirmDialog: React.ReactElement = (
        <ConfirmModal
            isOpen={isOpen}
            onClose={handleClose}
            onConfirm={handleConfirm}
            title={options.title}
            message={options.message}
            confirmText={options.confirmText}
            cancelText={options.cancelText}
            variant={options.variant ?? 'danger'}
        />
    );

    return { confirm, ConfirmDialog };
}
