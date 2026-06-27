// ═══════════════════════════════════════════════════════════════════════
// RCAWidget Tests
//
// P1-UX.7: RCA Widget in Device Overview
//   - Loading state
//   - Error state with retry
//   - "No RCA available" state
//   - Summary card with root cause, blast radius, affected devices
//   - Expand button opens modal
//   - Export PDF button
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';
import { RCAWidget } from '../RCAWidget';
import { api } from '../../../services/api';

// ── Mocks ────────────────────────────────────────────────────────────

vi.mock('../../../services/api', () => ({
    api: {
        getRCAGraph: vi.fn(),
    },
}));

// Mock modal — intercept the lazy-loaded RCAGraph inside modal
vi.mock('../RCAGraph', () => ({
    default: ({ deviceId }: { deviceId: string }) => (
        <div data-testid="full-rca-graph">RCAGraph for {deviceId}</div>
    ),
}));

// ── Test Data ────────────────────────────────────────────────────────

const MOCK_RCA_DATA = {
    nodes: [
        {
            id: 'nvr-1',
            data: {
                label: 'NVR Main',
                device_type: 'nvr',
                status: 'OFFLINE',
                is_root_cause: true,
                is_failed: true,
                is_healthy: false,
            },
            position: { x: 0, y: 0 },
        },
        {
            id: 'cam-1',
            data: {
                label: 'Camera Entrance',
                device_type: 'camera',
                status: 'OFFLINE',
                is_root_cause: false,
                is_failed: true,
                is_healthy: false,
            },
            position: { x: 200, y: 0 },
        },
        {
            id: 'cam-2',
            data: {
                label: 'Camera Parking',
                device_type: 'camera',
                status: 'DEGRADED',
                is_root_cause: false,
                is_failed: false,
                is_healthy: false,
            },
            position: { x: 400, y: 0 },
        },
    ],
    edges: [
        { id: 'e1', source: 'nvr-1', target: 'cam-1', animated: true },
        { id: 'e2', source: 'nvr-1', target: 'cam-2', animated: false },
    ],
    root_cause_id: 'nvr-1',
    failed_device_id: 'nvr-1',
    impact_description: 'NVR off — 2 cameras offline',
    recommendation: 'Restart NVR and check power supply',
    blast_radius: 2,
};

const EMPTY_RCA_DATA = {
    nodes: [],
    edges: [],
    root_cause_id: '',
    failed_device_id: '',
    impact_description: '',
    recommendation: '',
    blast_radius: 0,
};

// ── Wrapper ──────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
    return (
        <BrowserRouter>
            <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
        </BrowserRouter>
    );
}

// ── Tests ────────────────────────────────────────────────────────────

describe('RCAWidget', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    // P1-UX.7.1: Loading state
    it('shows loading state while fetching RCA data', () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockReturnValue(
            new Promise(() => {}),
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        // i18n returns the key as fallback when translation is missing
        expect(screen.getByText('rca_loading')).toBeTruthy();
        expect(document.querySelector('.animate-spin')).toBeTruthy();
    });

    // P1-UX.7.2: Error state with retry
    it('shows error state and allows retry', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockRejectedValue(
            new Error('Network error'),
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            expect(screen.getByText(/network error/i)).toBeTruthy();
        });

        // Click retry
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );
        fireEvent.click(screen.getByText(/retry/i));

        await waitFor(() => {
            expect(screen.getByText('NVR Main')).toBeTruthy();
        });
    });

    // P1-UX.7.3: "No RCA available" state
    it('shows empty state when no RCA data', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            EMPTY_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            // i18n key fallback
            expect(screen.getByText('rca_no_data')).toBeTruthy();
            expect(
                screen.getByText('rca_no_data_description'),
            ).toBeTruthy();
        });
    });

    // P1-UX.7.4: Summary card shows root cause
    it('displays root cause name and status', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            expect(screen.getByText('NVR Main')).toBeTruthy();
            expect(screen.getByText('OFFLINE')).toBeTruthy();
        });
    });

    // P1-UX.7.5: Summary card shows blast radius
    it('displays blast radius count', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            // Blast radius value rendered in the grid
            const blastEl = screen.getByText('2');
            expect(blastEl).toBeTruthy();
            // i18n key for "affected devices"
            expect(screen.getByText('affected_devices')).toBeTruthy();
        });
    });

    // P1-UX.7.6: Summary card shows affected device types
    it('displays affected device type badges', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            expect(screen.getByText('nvr')).toBeTruthy();
            expect(screen.getByText('camera')).toBeTruthy();
        });
    });

    // P1-UX.7.7: Summary card shows recommendation
    it('displays recommendation', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            expect(
                screen.getByText(/Restart NVR and check power supply/i),
            ).toBeTruthy();
        });
    });

    // P1-UX.7.8: Modal with full RCAGraph opens on expand click
    it('opens modal with full RCAGraph on expand click', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        // Wait for data to load
        await waitFor(() => {
            expect(screen.getByText('NVR Main')).toBeTruthy();
        });

        // Click expand button (aria-label uses i18n key fallback)
        const expandBtn = screen.getByLabelText('expand_rca_graph');
        fireEvent.click(expandBtn);

        await waitFor(() => {
            expect(screen.getByTestId('full-rca-graph')).toBeTruthy();
        });
    });

    // P1-UX.7.9: Failed device count (2 failed in mock data: nvr-1 + cam-1)
    it('shows count of failed devices', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            // Format: "{count} {t('failed')}" = "2 failed"
            expect(screen.getByText('2')).toBeTruthy();
            expect(screen.getByText(/failed/i)).toBeTruthy();
        });
    });

    // P1-UX.7.10: Impact description
    it('displays impact description', async () => {
        (api.getRCAGraph as ReturnType<typeof vi.fn>).mockResolvedValue(
            MOCK_RCA_DATA,
        );

        render(<RCAWidget deviceId="dev-1" />, { wrapper: Wrapper });

        await waitFor(() => {
            expect(
                screen.getByText(/NVR off.*2 cameras offline/i),
            ).toBeTruthy();
        });
    });
});
