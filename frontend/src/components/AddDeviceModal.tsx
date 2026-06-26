import React, { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Modal, Button, Input, Select, useToast } from './ui';
import { ConnectionType } from '../types';
import { useSites, useCreateDevice } from '../hooks/useApiQuery';
import { generateUUID } from '../utils/uuid';
import { addDeviceSchema, AddDeviceFormData } from '../lib/validations';

interface Props {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
}

const P2P_BRANDS = [
    { value: 'hikvision', label: 'Hikvision' },
    { value: 'dahua', label: 'Dahua' },
    { value: 'reolink', label: 'Reolink' },
    { value: 'xiongmai', label: 'Xiongmai' },
    { value: 'ezviz', label: 'EZVIZ' },
] as const;

const CONNECTION_TYPES = [
    { value: 'ip', labelKey: 'type_ip_camera' },
    { value: 'p2p', labelKey: 'type_p2p_camera' },
    { value: 'snmp', labelKey: 'type_snmp_device' },
    { value: 'syslog', labelKey: 'type_syslog_device' },
    { value: 'alarm', labelKey: 'type_alarm_receiver' },
] as const;

const SNMP_VERSIONS = [
    { value: 'v1', label: 'SNMP v1' },
    { value: 'v2c', label: 'SNMP v2c' },
    { value: 'v3', label: 'SNMP v3' },
] as const;

const ALARM_PROTOCOLS = [
    { value: 'http', label: 'HTTP XML' },
    { value: 'sip', label: 'SIP' },
    { value: 'xml', label: 'XML' },
] as const;

/** Бренды, требующие cloud-credentials */
const NEEDS_CLOUD_CREDS = new Set(['dahua', 'reolink', 'ezviz']);

export const AddDeviceModal: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
    const { t } = useTranslation();
    const toast = useToast();
    const { data: rawSites = [] } = useSites();
    const createDevice = useCreateDevice();

    const sites = useMemo(() => rawSites.map((s: Record<string, any>) => ({
        id: s.id,
        name: s.name || t('unnamed_site'),
        address: s.address || '',
        city: s.city || '',
        organization: s.organization || '',
        latitude: s.latitude || 0,
        longitude: s.longitude || 0,
        status: s.status || 'active',
        lastSync: s.last_sync || new Date().toISOString(),
    })), [rawSites, t]);

    const siteOptions = sites.map(site => ({ value: site.id, label: site.name }));
    const noSites = sites.length === 0;

    const {
        register,
        handleSubmit,
        watch,
        reset,
        formState: { errors, isValid, isSubmitting },
    } = useForm<AddDeviceFormData>({
        resolver: zodResolver(addDeviceSchema),
        mode: 'onChange',
        defaultValues: {
            name: '',
            siteId: '',
            connectionType: 'ip',
            model: '',
            ipAddress: '',
            p2pBrand: 'hikvision',
            p2pSerial: '',
            p2pSecurityCode: '',
            p2pCloudUser: '',
            p2pCloudPass: '',
            snmpCommunity: 'public',
            snmpVersion: 'v2c',
            syslogPort: 514,
            alarmProtocol: 'http',
        },
    });

    const connectionType = watch('connectionType');
    const p2pBrand = watch('p2pBrand');
    const needsCredentials = p2pBrand ? NEEDS_CLOUD_CREDS.has(p2pBrand) : false;

    // Reset form on modal close
    useEffect(() => {
        if (!isOpen) {
            reset();
        } else if (sites.length > 0) {
            // Auto-select first site when modal opens
            const currentSiteId = watch('siteId');
            if (!currentSiteId) {
                reset({ siteId: sites[0].id });
            }
        }
    }, [isOpen, sites, reset, watch]);

    /** Translate Zod error message keys */
    const translateError = (key: string | undefined): string | undefined => {
        if (!key) return undefined;
        // If it's an i18n key, translate it; otherwise return as-is
        return key.startsWith('validation.') ? t(key) : key;
    };

    const onSubmit = async (data: AddDeviceFormData) => {
        if (noSites) {
            toast.error(t('no_sites_available'));
            return;
        }

        try {
            const selectedSite = sites.find(s => s.id === data.siteId);
            const siteName = selectedSite?.name || t('unknown_device');

            // Формируем payload для API (snake_case поля как ожидает бэкенд)
            const payload: Record<string, any> = {
                device_id: generateUUID(),
                name: data.name,
                device_type: data.connectionType === 'ip' ? 'camera' : 'switch',
                status: 'ONLINE',
                connection_type: data.connectionType,
                asset_class: 'internal',
                site_id: data.siteId,
            };

            // IP-адрес
            if (data.connectionType === 'ip' && data.ipAddress) {
                payload.ip_address = data.ipAddress;
            }

            // Специфичные поля
            if (data.connectionType === 'p2p') {
                payload.p2p_brand = data.p2pBrand;
                payload.p2p_serial = data.p2pSerial;
                payload.p2p_security_code = data.p2pSecurityCode;
                if (data.p2pCloudUser) payload.p2p_cloud_user = data.p2pCloudUser;
                if (data.p2pCloudPass) payload.p2p_cloud_pass = data.p2pCloudPass;
            } else if (data.connectionType === 'snmp') {
                payload.snmp_community = data.snmpCommunity;
                payload.snmp_version = data.snmpVersion;
            } else if (data.connectionType === 'syslog') {
                payload.syslog_port = data.syslogPort;
            } else if (data.connectionType === 'alarm') {
                payload.alarm_protocol = data.alarmProtocol;
            }

            await createDevice.mutateAsync(payload as any);

            toast.success(t('device_added_success'));
            onSuccess?.();
            onClose();
        } catch (err: any) {
            console.error(err);
            toast.error(err.message || t('add_device_failed'));
        }
    };

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
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
                {/* Connection Type */}
                <Select
                    label={`${t('connection_type')} *`}
                    options={CONNECTION_TYPES.map(ct => ({
                        value: ct.value,
                        label: t(ct.labelKey),
                    }))}
                    error={translateError(errors.connectionType?.message)}
                    {...register('connectionType', {
                        onChange: (e) => {
                            // Reset IP when switching away from IP
                            if (e.target.value !== 'ip') {
                                // reset will happen on next render via watched value
                            }
                        },
                    })}
                />

                {/* Device Name */}
                <Input
                    label={`${t('device_name')} *`}
                    placeholder={t('device_name')}
                    error={translateError(errors.name?.message)}
                    {...register('name')}
                />

                {/* Site */}
                <Select
                    label={`${t('site')} *`}
                    options={siteOptions}
                    error={translateError(errors.siteId?.message)}
                    {...register('siteId')}
                />

                {/* IP Address (conditional) */}
                {connectionType === 'ip' && (
                    <Input
                        label={`${t('ip_address')} *`}
                        placeholder={t('ip_placeholder')}
                        error={translateError(errors.ipAddress?.message)}
                        {...register('ipAddress')}
                    />
                )}

                {/* Model */}
                <Input
                    label={t('model')}
                    placeholder={t('model_placeholder')}
                    {...register('model')}
                />

                {/* P2P Fields (conditional) */}
                {connectionType === 'p2p' && (
                    <>
                        <Select
                            label={t('brand')}
                            options={P2P_BRANDS.map(b => ({ value: b.value, label: b.label }))}
                            {...register('p2pBrand')}
                        />
                        <Input
                            label={`${t('serial_number')} *`}
                            placeholder={t('serial_placeholder')}
                            error={translateError(errors.p2pSerial?.message)}
                            {...register('p2pSerial')}
                        />
                        <Input
                            label={t('security_code')}
                            type="password"
                            placeholder={t('security_code_placeholder')}
                            {...register('p2pSecurityCode')}
                        />
                        {needsCredentials && (
                            <>
                                <Input
                                    label={t('cloud_username')}
                                    {...register('p2pCloudUser')}
                                />
                                <Input
                                    label={t('cloud_password')}
                                    type="password"
                                    {...register('p2pCloudPass')}
                                />
                            </>
                        )}
                    </>
                )}

                {/* SNMP Fields (conditional) */}
                {connectionType === 'snmp' && (
                    <>
                        <Input
                            label={t('snmp_community')}
                            {...register('snmpCommunity')}
                        />
                        <Select
                            label={t('snmp_version')}
                            options={SNMP_VERSIONS.map(v => ({ value: v.value, label: v.label }))}
                            {...register('snmpVersion')}
                        />
                    </>
                )}

                {/* Syslog Fields (conditional) */}
                {connectionType === 'syslog' && (
                    <Input
                        label={t('syslog_port')}
                        type="number"
                        error={translateError(errors.syslogPort?.message)}
                        {...register('syslogPort', { valueAsNumber: true })}
                    />
                )}

                {/* Alarm Fields (conditional) */}
                {connectionType === 'alarm' && (
                    <Select
                        label={t('alarm_protocol')}
                        options={ALARM_PROTOCOLS.map(p => ({ value: p.value, label: p.label }))}
                        {...register('alarmProtocol')}
                    />
                )}

                {/* Actions */}
                <div className="flex justify-end gap-3 pt-4">
                    <Button type="button" variant="outline" onClick={onClose}>
                        {t('cancel')}
                    </Button>
                    <Button
                        type="submit"
                        loading={isSubmitting}
                        disabled={!isValid || isSubmitting}
                    >
                        {t('add_device')}
                    </Button>
                </div>
            </form>
        </Modal>
    );
};
