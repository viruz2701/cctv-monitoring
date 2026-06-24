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
    addSite: (site: Site) => Promise<void>;
    updateSite: (id: string, updates: Partial<Site>) => Promise<void>;
    deleteSite: (id: string) => Promise<void>;
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
            const [devs, siteList] = await Promise.all([
                api.getDevices(),
                api.getSites(),
            ]);
            // OWASP ASVS V5: Input validation — API может вернуть { devices: [...] } вместо прямого массива
            const devsArray: any[] = Array.isArray(devs) ? devs : (devs && typeof devs === 'object' && 'devices' in devs ? (devs as any).devices : []);
            const siteArray: any[] = Array.isArray(siteList) ? siteList : (siteList && typeof siteList === 'object' && 'sites' in siteList ? (siteList as any).sites : []);
            const mapped: Device[] = devsArray.map((d: any) => ({
                id: d.device_id,
                name: d.name || d.device_id,
                siteId: d.site_id || 'site-default',
                siteName: d.location || 'Unknown',
                type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
                status: (d.status || 'offline').toLowerCase(),
                health: d.status === 'online' ? 'healthy' : 'faulty',
                recordingStatus: 'recording',
                lastSeen: d.last_seen || new Date().toISOString(),
                ipAddress: '',
                model: d.vendor_type || '',
                firmware: '',
                owner_id: d.owner_id,
            }));
            const filtered = user?.role === 'owner' ? mapped.filter(d => d.owner_id === user.id) : mapped;
            setDevices(filtered);

            // Map sites from API — OWASP ASVS V5: input validation
            const mappedSites: Site[] = siteArray.map((s: any) => ({
                id: s.id,
                name: s.name || 'Unnamed',
                address: s.address || '',
                city: s.city || '',
                organization: s.organization || '',
                latitude: s.latitude || 0,
                longitude: s.longitude || 0,
                status: s.status || 'active',
                lastSync: s.last_sync || new Date().toISOString(),
            }));
            setSites(mappedSites.length > 0 ? mappedSites : []);
        } catch (err) {
            console.error('Failed to load data:', err);
            // If API fails, try to at least show something
            if (sites.length === 0) {
                setSites([{ id: 'site-default', name: 'Default Site', address: '', city: '', status: 'active', lastSync: new Date().toISOString() }]);
            }
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        if (token) loadData();
    }, [token, user]);

    const refresh = loadData;

    const addDevice = (device: Device) => setDevices(prev => [...prev, device]);
    const updateDevice = (id: string, updates: Partial<Device>) => setDevices(prev => prev.map(d => d.id === id ? { ...d, ...updates } : d));
    const deleteDevice = (id: string) => setDevices(prev => prev.filter(d => d.id !== id));

    const addSite = async (site: Site) => {
        try {
            const created = await api.createSite({
                name: site.name,
                address: site.address,
                city: site.city,
                status: site.status,
            });
            const newSite: Site = {
                id: created.id,
                name: created.name || site.name,
                address: created.address || site.address,
                city: created.city || site.city,
                organization: site.organization || '',
                latitude: site.latitude || 0,
                longitude: site.longitude || 0,
                status: created.status || site.status,
                lastSync: new Date().toISOString(),
            };
            setSites(prev => [...prev, newSite]);
        } catch (err) {
            console.error('Failed to create site via API, adding locally:', err);
            setSites(prev => [...prev, site]);
        }
    };

    const updateSite = async (id: string, updates: Partial<Site>) => {
        try {
            await api.updateSite(id, updates);
            setSites(prev => prev.map(s => s.id === id ? { ...s, ...updates } : s));
        } catch (err) {
            console.error('Failed to update site via API:', err);
            setSites(prev => prev.map(s => s.id === id ? { ...s, ...updates } : s));
        }
    };

    const deleteSite = async (id: string) => {
        try {
            await api.deleteSite(id);
            setSites(prev => prev.filter(s => s.id !== id));
        } catch (err) {
            console.error('Failed to delete site via API:', err);
            setSites(prev => prev.filter(s => s.id !== id));
        }
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
