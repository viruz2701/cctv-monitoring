import { useTickets } from './TicketsContext';
import { useAlerts } from './AlertsContext';
import { useDevicesSites } from './DevicesSitesContext';
import { useUsers } from './UsersContext';
import { useSettings } from './SettingsContext';

// Re-export hooks for backward compatibility
export { useTickets } from './TicketsContext';
export { useAlerts } from './AlertsContext';
export { useDevicesSites } from './DevicesSitesContext';
export { useUsers } from './UsersContext';
export { useSettings } from './SettingsContext';
export { useReports } from './ReportsContext';

// ── Backward-compatible façade ───────────────────────────────────────────
// Composes all 5 domain hooks into a single object matching the old API.
// Consumers should migrate to domain-specific hooks for optimal performance.

export function useData() {
    return {
        ...useTickets(),
        ...useAlerts(),
        ...useDevicesSites(),
        ...useUsers(),
        ...useSettings(),
    };
}

// Deprecated DataProvider - generic wrapper if needed, but App.tsx now composes them manually
// We can remove this export if no other files import DataProvider directly except App.tsx (which we updated)
export function DataProvider({ children }: { children: React.ReactNode }) {
    console.warn('DataProvider is deprecated. Please use composed providers in App.tsx');
    return <>{children}</>;
}
