import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Button, Input, Select, useToast } from './ui';
import { ConnectionType, P2PRegistrationForm, Device } from '../types';
import { useSites, useCreateDevice, useUpdateDevice } from '../hooks/useApiQuery';
import { generateUUID } from '../utils/uuid';

interface Props {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
}

export const AddDeviceModal: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
    const { t } = useTranslation();
    const toast = useToast();
    const { data: rawSites = [] } = useSites();
    const createDevice = useCreateDevice();
    const updateDeviceMut = useUpdateDevice();

    const sites = useMemo(() => rawSites.map((s: Record<string, any>) => ({
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        organization: s.organization || '',
        latitude: s.latitude || 0,
        longitude: s.longitude || 0,
        status: s.status || 'active',
        lastSync: s.last_sync || new Date().toISOString(),
    })), [rawSites]);
    const [loading, setLoading] = useState(false);
    const [connectionType, setConnectionType] = useState<ConnectionType>('ip');
    
    // Общие поля
    const [name, setName] = useState('');
    const [siteId, setSiteId] = useState('');
    const [ipAddress, setIpAddress] = useState('');
    const [model, setModel] = useState('');

    // P2P поля
    const [p2pBrand, setP2pBrand] = useState('hikvision');
    const [p2pSerial, setP2pSerial] = useState('');
    const [p2pSecurityCode, setP2pSecurityCode] = useState('');
    const [p2pCloudUser, setP2pCloudUser] = useState('');
    const [p2pCloudPass, setP2pCloudPass] = useState('');

    // SNMP поля
    const [snmpCommunity, setSnmpCommunity] = useState('public');
    const [snmpVersion, setSnmpVersion] = useState<'v1'|'v2c'|'v3'>('v2c');

    // Syslog поля
    const [syslogPort, setSyslogPort] = useState(514);

    // Alarm поля
    const [alarmProtocol, setAlarmProtocol] = useState<'http'|'sip'|'xml'>('http');

    // Сброс формы при открытии/закрытии
    useEffect(() => {
        if (!isOpen) {
            // Сброс при закрытии
            setName('');
            setSiteId('');
            setIpAddress('');
            setModel('');
            setP2pBrand('hikvision');
            setP2pSerial('');
            setP2pSecurityCode('');
            setP2pCloudUser('');
            setP2pCloudPass('');
            setSnmpCommunity('public');
            setSnmpVersion('v2c');
            setSyslogPort(514);
            setAlarmProtocol('http');
            setConnectionType('ip');
            setLoading(false);
        } else {
            // При открытии, если есть хоть один сайт, выбираем первый по умолчанию
            if (sites.length > 0 && !siteId) {
                setSiteId(sites[0].id);
            }
        }
    }, [isOpen, sites]);

    const siteOptions = sites.map(site => ({ value: site.id, label: site.name }));
    const noSites = sites.length === 0;

    const isRequiredFilled = (): boolean => {
        if (!name || !siteId) return false;
        if (connectionType === 'ip' && !ipAddress) return false;
        if (connectionType === 'p2p' && !p2pSerial) return false;
        return true;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (noSites) {
            toast.error(t('no_sites_available'));
            return;
        }
        if (!isRequiredFilled()) {
            toast.error(t('please_fill_required_fields'));
            return;
        }
        setLoading(true);
        try {
            const selectedSite = sites.find(s => s.id === siteId);
            const siteName = selectedSite?.name || 'Unknown';

            // Базовое устройство
            const newDevice: Device = {
                id: `dev-${generateUUID()}`,
                name: name,
                siteId: siteId,
                siteName: siteName,
                type: connectionType === 'ip' ? 'camera' : 'switch', // базовый тип
                status: 'online',
                health: 'healthy',
                recordingStatus: 'recording',
                lastSeen: new Date().toISOString(),
                ipAddress: connectionType === 'ip' ? ipAddress : '',
                model: model || (connectionType === 'p2p' ? p2pBrand : connectionType),
                firmware: '1.0.0',
                connectionType: connectionType,
            };

            // Добавляем специфичные поля в зависимости от типа
            if (connectionType === 'p2p') {
                newDevice.p2p_brand = p2pBrand;
                newDevice.p2p_serial = p2pSerial;
                newDevice.p2p_security_code = p2pSecurityCode;
                if (p2pCloudUser) newDevice.p2p_cloud_user = p2pCloudUser;
                if (p2pCloudPass) newDevice.p2p_cloud_pass = p2pCloudPass;
                newDevice.cloud_status = 'unknown';
                // Для P2P также можно вызвать регистрацию через шлюз, но не блокируем добавление
                // Пока просто добавляем устройство локально
            } else if (connectionType === 'snmp') {
                newDevice.snmp_community = snmpCommunity;
                newDevice.snmp_version = snmpVersion;
            } else if (connectionType === 'syslog') {
                newDevice.syslog_port = syslogPort;
            } else if (connectionType === 'alarm') {
                newDevice.alarm_protocol = alarmProtocol;
            }

            // Добавляем устройство через API
            await createDevice.mutateAsync(newDevice);

            toast.success(t('device_added_success') || 'Device added successfully');
            onSuccess?.();
            onClose(); // закрываем модальное окно
        } catch (err: any) {
            console.error(err);
            toast.error(err.message || t('add_device_failed'));
        } finally {
            setLoading(false);
        }
    };

    const needsCredentials = ['dahua', 'reolink', 'ezviz'].includes(p2pBrand);

    // Если сайтов нет, показываем сообщение и блокируем форму
    if (noSites) {
        return (
            <Modal isOpen={isOpen} onClose={onClose} title={t('add_device')} size="lg">
                <div className="p-4 text-center">
                    <p className="text-red-500">{t('no_sites_message')}</p>
                    <Button className="mt-4" onClick={onClose}>{t('close')}</Button>
                </div>
            </Modal>
        );
    }

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={t('add_device')} size="lg">
            <form onSubmit={handleSubmit} className="space-y-4">
                <Select
                    label={t('connection_type')}
                    options={[
                        { value: 'ip', label: t('type_ip_camera') },
                        { value: 'p2p', label: t('type_p2p_camera') },
                        { value: 'snmp', label: t('type_snmp_device') },
                        { value: 'syslog', label: t('type_syslog_device') },
                        { value: 'alarm', label: t('type_alarm_receiver') },
                    ]}
                    value={connectionType}
                    onChange={(e) => {
                        setConnectionType(e.target.value as ConnectionType);
                        if (e.target.value !== 'ip') setIpAddress('');
                    }}
                />

                <Input
                    label={t('device_name')}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                />

                <Select
                    label={t('site')}
                    options={siteOptions}
                    value={siteId}
                    onChange={(e) => setSiteId(e.target.value)}
                    required
                />

                {connectionType === 'ip' && (
                    <Input
                        label={t('ip_address')}
                        value={ipAddress}
                        onChange={(e) => setIpAddress(e.target.value)}
                        required
                        placeholder="e.g., 192.168.1.100"
                    />
                )}

                <Input
                    label={t('model')}
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder="e.g., DS-2CD2143G2"
                />

                {connectionType === 'p2p' && (
                    <>
                        <Select
                            label={t('brand')}
                            options={[
                                { value: 'hikvision', label: 'Hikvision' },
                                { value: 'dahua', label: 'Dahua' },
                                { value: 'reolink', label: 'Reolink' },
                                { value: 'xiongmai', label: 'Xiongmai' },
                                { value: 'ezviz', label: 'EZVIZ' },
                            ]}
                            value={p2pBrand}
                            onChange={(e) => setP2pBrand(e.target.value)}
                        />
                        <Input
                            label={t('serial_number')}
                            placeholder="e.g., 95270DSD7FFRVTAS7"
                            value={p2pSerial}
                            onChange={(e) => setP2pSerial(e.target.value)}
                            required
                        />
                        <Input
                            label={t('security_code')}
                            type="password"
                            placeholder={t('security_code_placeholder')}
                            value={p2pSecurityCode}
                            onChange={(e) => setP2pSecurityCode(e.target.value)}
                        />
                        {needsCredentials && (
                            <>
                                <Input
                                    label={t('cloud_username')}
                                    value={p2pCloudUser}
                                    onChange={(e) => setP2pCloudUser(e.target.value)}
                                />
                                <Input
                                    label={t('cloud_password')}
                                    type="password"
                                    value={p2pCloudPass}
                                    onChange={(e) => setP2pCloudPass(e.target.value)}
                                />
                            </>
                        )}
                    </>
                )}

                {connectionType === 'snmp' && (
                    <>
                        <Input
                            label={t('snmp_community')}
                            value={snmpCommunity}
                            onChange={(e) => setSnmpCommunity(e.target.value)}
                        />
                        <Select
                            label={t('snmp_version')}
                            options={[
                                { value: 'v1', label: 'SNMP v1' },
                                { value: 'v2c', label: 'SNMP v2c' },
                                { value: 'v3', label: 'SNMP v3' },
                            ]}
                            value={snmpVersion}
                            onChange={(e) => setSnmpVersion(e.target.value as any)}
                        />
                    </>
                )}

                {connectionType === 'syslog' && (
                    <Input
                        label={t('syslog_port')}
                        type="number"
                        value={syslogPort}
                        onChange={(e) => setSyslogPort(Number(e.target.value))}
                    />
                )}

                {connectionType === 'alarm' && (
                    <Select
                        label={t('alarm_protocol')}
                        options={[
                            { value: 'http', label: 'HTTP XML' },
                            { value: 'sip', label: 'SIP' },
                            { value: 'xml', label: 'XML' },
                        ]}
                        value={alarmProtocol}
                        onChange={(e) => setAlarmProtocol(e.target.value as any)}
                    />
                )}

                <div className="flex justify-end gap-3 pt-4">
                    <Button type="button" variant="outline" onClick={onClose}>
                        {t('cancel')}
                    </Button>
                    <Button
                        type="submit"
                        loading={loading}
                        disabled={!isRequiredFilled()}
                    >
                        {t('add_device')}
                    </Button>
                </div>
            </form>
        </Modal>
    );
};