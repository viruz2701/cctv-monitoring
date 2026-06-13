import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api } from '../services/api';
import { useAuth } from '../hooks/useAuth';
import type { Device, Site } from '../types';

interface DevicesSitesContextType {
    devices: Device[];
    sites: Site[];
    loading: boolean;
    refresh: () => Promise<void>;
    addDevice: (device: Device) => void;
    updateDevice: (id: string, updates: Partial<Device>) => void;
    deleteDevice: (id: string) => void;
    addSite: (site: Site) => void;
    updateSite: (id: string, updates: Partial<Site>) => void;
    deleteSite: (id: string) => void;
}

const DevicesSitesContext = createContext<DevicesSitesContextType | undefined>(undefined);

export function DevicesSitesProvider({ children }: { children: ReactNode }) {
    const { user, token } = useAuth();
    const [devices, setDevices] = useState<Device[]>([]);
    const [sites, setSites] = useState<Site[]>([]);
    const [loading, setLoading] = useState(true);

    const loadData = async () => {
        if (!token) return;
        setLoading(true);
        try {
            const devs = await api.getDevices();
            const mapped: Device[] = devs.map((d: any) => ({
                id: d.device_id,
                name: d.name || d.device_id,
                siteId: 'site-1',
                siteName: d.location || 'Unknown',
                type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
                status: d.status.toLowerCase(),
                health: d.status === 'online' ? 'healthy' : 'faulty',
                recordingStatus: 'recording',
                lastSeen: d.last_seen,
                ipAddress: '',
                model: d.vendor_type,
                firmware: '',
                owner_id: d.owner_id,
            }));
            const filtered = user?.role === 'owner' ? mapped.filter(d => d.owner_id === user.id) : mapped;
            setDevices(filtered);
            setSites([{ id: 'site-1', name: 'Main Site', address: '', city: '', status: 'active', lastSync: new Date().toISOString() }]);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (token) loadData();
    }, [token, user]);

    const refresh = loadData;

    // Заглушки для CRUD (позже можно заменить на реальные вызовы API)
    const addDevice = (device: Device) => setDevices(prev => [...prev, device]);
    const updateDevice = (id: string, updates: Partial<Device>) => setDevices(prev => prev.map(d => d.id === id ? { ...d, ...updates } : d));
    const deleteDevice = (id: string) => setDevices(prev => prev.filter(d => d.id !== id));
    const addSite = (site: Site) => setSites(prev => [...prev, site]);
    const updateSite = (id: string, updates: Partial<Site>) => setSites(prev => prev.map(s => s.id === id ? { ...s, ...updates } : s));
    const deleteSite = (id: string) => setSites(prev => prev.filter(s => s.id !== id));

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