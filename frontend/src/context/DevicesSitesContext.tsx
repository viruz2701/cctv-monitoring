// ═══════════════════════════════════════════════════════════════════════
// DevicesSitesContext — Bridge to React Query Hooks (ARCH-02)
//
// Обеспечивает обратную совместимость. Новый код ДОЛЖЕН использовать
// хуки из useApiQuery напрямую.
//
// Миграция:
//   Было:  import { useDevicesSites } from './context/DevicesSitesContext'
//   Стало: import { useDevices, useSites } from '../hooks/useApiQuery'
//
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, ReactNode } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useDevices, useSites, useCreateSite, useUpdateSite, useDeleteSite } from '../hooks/useApiQuery';
import type { Device, Site } from '../types';

interface DevicesSitesContextType {
    devices: Device[];
    sites: Site[];
    loading: boolean;
    refresh: () => void;
    addDevice: (device: Device) => void;
    updateDevice: (id: string, updates: Partial<Device>) => void;
    deleteDevice: (id: string) => void;
    addSite: (site: Site) => Promise<void>;
    updateSite: (id: string, updates: Partial<Site>) => Promise<void>;
    deleteSite: (id: string) => Promise<void>;
}

const DevicesSitesContext = createContext<DevicesSitesContextType | undefined>(undefined);

// ═══ Helper: маппинг API Device → UI Device ═══
function mapAPIDeviceToUI(d: any): Device {
    return {
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: d.site_id || 'site-default',
        siteName: d.location || 'Unknown',
        type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
        status: (d.status || 'offline').toLowerCase() as Device['status'],
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording',
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: d.vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    };
}

function mapAPISiteToUI(s: any): Site {
    return {
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        organization: s.organization || '',
        latitude: s.latitude || 0,
        longitude: s.longitude || 0,
        status: s.status || 'active',
        lastSync: s.last_sync || new Date().toISOString(),
    };
}

export function DevicesSitesProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth();

    // Используем React Query hooks (ARCH-02)
    const { data: rawDevices = [], isLoading: devicesLoading, refetch: refetchDevices } = useDevices();
    const { data: rawSites = [], isLoading: sitesLoading, refetch: refetchSites } = useSites();

    const createSiteMutation = useCreateSite();
    const updateSiteMutation = useUpdateSite();
    const deleteSiteMutation = useDeleteSite();

    const loading = devicesLoading || sitesLoading;

    // OWASP ASVS V5: Input validation
    const devsArray: any[] = Array.isArray(rawDevices)
        ? rawDevices
        : (rawDevices && typeof rawDevices === 'object' && 'devices' in rawDevices
            ? (rawDevices as any).devices : []);

    const siteArray: any[] = Array.isArray(rawSites)
        ? rawSites
        : (rawSites && typeof rawSites === 'object' && 'sites' in rawSites
            ? (rawSites as any).sites : []);

    const apiDevices: Device[] = devsArray.map(mapAPIDeviceToUI);
    const devices = user?.role === 'owner'
        ? apiDevices.filter(d => d.owner_id === user.id)
        : apiDevices;

    const sites: Site[] = siteArray.map(mapAPISiteToUI);

    const refresh = () => {
        refetchDevices();
        refetchSites();
    };

    // ═══ Client-side mutations (optimistic updates) ═══
    // Эти операции используются редко, поэтому здесь они остаются
    // как заглушки. Для production нужно реализовать через React Query mutations.

    const addDevice = (_device: Device) => {
        console.warn('addDevice: use useCreateDevice() from hooks/useApiQuery instead');
        refetchDevices();
    };

    const updateDevice = (_id: string, _updates: Partial<Device>) => {
        console.warn('updateDevice: use useUpdateDevice() from hooks/useApiQuery instead');
        refetchDevices();
    };

    const deleteDevice = (_id: string) => {
        console.warn('deleteDevice: use useDeleteDevice() from hooks/useApiQuery instead');
        refetchDevices();
    };

    const addSite = async (site: Site) => {
        await createSiteMutation.mutateAsync({
            name: site.name,
            address: site.address,
            city: site.city,
            status: site.status,
        });
    };

    const updateSite = async (id: string, updates: Partial<Site>) => {
        await updateSiteMutation.mutateAsync({ id, updates });
    };

    const deleteSite = async (id: string) => {
        await deleteSiteMutation.mutateAsync(id);
    };

    return (
        <DevicesSitesContext.Provider value={{ devices, sites, loading, refresh, addDevice, updateDevice, deleteDevice, addSite, updateSite, deleteSite }}>
            {children}
        </DevicesSitesContext.Provider>
    );
}

export function useDevicesSites() {
    const ctx = useContext(DevicesSitesContext);
    if (!ctx) throw new Error('useDevicesSites must be used within DevicesSitesProvider');
    return ctx;
}
