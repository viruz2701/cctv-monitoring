import React, { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Modal, Button, Input, Select, useToast } from './ui';
import { useSites, useCreateDevice } from '../hooks/useApiQuery';
import { generateUUID } from '../utils/uuid';
import {
  addDeviceSchema,
  ADD_DEVICE_DEFAULTS,
  AddDeviceFormData,
} from '../lib/validations/device';

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
  { value: 'onvif', labelKey: 'type_onvif_camera' },
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

/**
 * Translate Zod error message keys using i18n.
 * Supports both `validation.*` keys and raw strings.
 */
const translateError = (t: (key: string) => string, message?: string): string | undefined => {
  if (!message) return undefined;
  return message.startsWith('validation.') ? t(message) : message;
};

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
    defaultValues: ADD_DEVICE_DEFAULTS,
  });

  const connectionType = watch('connectionType');
  const p2pBrand = watch('p2pBrand');
  const needsCredentials = p2pBrand ? NEEDS_CLOUD_CREDS.has(p2pBrand) : false;

  // Reset form on modal close / auto-select first site
  useEffect(() => {
    if (!isOpen) {
      reset(ADD_DEVICE_DEFAULTS);
    } else if (sites.length > 0) {
      const currentSiteId = watch('siteId');
      if (!currentSiteId) {
        reset({ ...ADD_DEVICE_DEFAULTS, siteId: sites[0].id });
      }
    }
  }, [isOpen, sites, reset, watch]);

  /**
   * Helper: get translated error for a field.
   */
  const fieldError = (field: keyof AddDeviceFormData): string | undefined => {
    return translateError(t, errors[field]?.message);
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
        device_type: data.connectionType === 'ip' || data.connectionType === 'onvif'
          ? 'camera'
          : 'switch',
        status: 'ONLINE',
        connection_type: data.connectionType,
        asset_class: 'internal',
        site_id: data.siteId,
      };

      // IP-адрес (IP + ONVIF)
      if ((data.connectionType === 'ip' || data.connectionType === 'onvif') && data.ipAddress) {
        payload.ip_address = data.ipAddress;
      }

      // Специфичные поля
      if (data.connectionType === 'p2p') {
        payload.p2p_brand = data.p2pBrand;
        payload.p2p_serial = data.p2pSerial;
        payload.p2p_security_code = data.p2pSecurityCode;
        if (data.p2pCloudUser) payload.p2p_cloud_user = data.p2pCloudUser;
        if (data.p2pCloudPass) payload.p2p_cloud_pass = data.p2pCloudPass;
      } else if (data.connectionType === 'onvif') {
        payload.onvif_username = data.onvifUsername;
        payload.onvif_password = data.onvifPassword;
      } else if (data.connectionType === 'snmp') {
        payload.snmp_community = data.snmpCommunity;
        payload.snmp_version = data.snmpVersion;
      } else if (data.connectionType === 'syslog') {
        payload.syslog_port = data.syslogPort;
      } else if (data.connectionType === 'alarm') {
        payload.alarm_protocol = data.alarmProtocol;
      }

      // Location fields
      if (data.location) payload.location = data.location;
      if (data.latitude !== undefined) payload.latitude = data.latitude;
      if (data.longitude !== undefined) payload.longitude = data.longitude;

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
        {/* ═══ Connection Type ═══════════════════════════════════════════ */}
        <Select
          label={`${t('connection_type')} *`}
          options={CONNECTION_TYPES.map(ct => ({
            value: ct.value,
            label: t(ct.labelKey),
          }))}
          error={fieldError('connectionType')}
          {...register('connectionType')}
        />

        {/* ═══ Device Name ═══════════════════════════════════════════════ */}
        <Input
          label={`${t('device_name')} *`}
          placeholder={t('device_name')}
          error={fieldError('name')}
          {...register('name')}
        />

        {/* ═══ Site ══════════════════════════════════════════════════════ */}
        <Select
          label={`${t('site')} *`}
          options={siteOptions}
          error={fieldError('siteId')}
          {...register('siteId')}
        />

        {/* ═══ IP Address (conditional: IP / ONVIF) ═════════════════════ */}
        {(connectionType === 'ip' || connectionType === 'onvif') && (
          <>
            <Input
              label={`${t('ip_address')} *`}
              placeholder={t('ip_placeholder')}
              error={fieldError('ipAddress')}
              {...register('ipAddress')}
            />

            {/* Port (optional) */}
            <Input
              label={t('port')}
              type="number"
              placeholder="e.g., 554"
              error={fieldError('port')}
              {...register('port', { valueAsNumber: true })}
            />
          </>
        )}

        {/* ═══ ONVIF Credentials (conditional) ══════════════════════════ */}
        {connectionType === 'onvif' && (
          <>
            <Input
              label={`${t('onvif_username')} *`}
              placeholder="admin"
              error={fieldError('onvifUsername')}
              {...register('onvifUsername')}
            />
            <Input
              label={`${t('onvif_password')} *`}
              type="password"
              placeholder="••••••••"
              error={fieldError('onvifPassword')}
              {...register('onvifPassword')}
            />
          </>
        )}

        {/* ═══ Model ════════════════════════════════════════════════════ */}
        <Input
          label={t('model')}
          placeholder={t('model_placeholder')}
          error={fieldError('model')}
          {...register('model')}
        />

        {/* ═══ P2P Fields (conditional) ════════════════════════════════ */}
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
              error={fieldError('p2pSerial')}
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

        {/* ═══ SNMP Fields (conditional) ════════════════════════════════ */}
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

        {/* ═══ Syslog Fields (conditional) ══════════════════════════════ */}
        {connectionType === 'syslog' && (
          <Input
            label={`${t('syslog_port')} *`}
            type="number"
            error={fieldError('syslogPort')}
            {...register('syslogPort', { valueAsNumber: true })}
          />
        )}

        {/* ═══ Alarm Fields (conditional) ═══════════════════════════════ */}
        {connectionType === 'alarm' && (
          <Select
            label={t('alarm_protocol')}
            options={ALARM_PROTOCOLS.map(p => ({ value: p.value, label: p.label }))}
            {...register('alarmProtocol')}
          />
        )}

        {/* ═══ Location (optional) ══════════════════════════════════════ */}
        <details className="group">
          <summary className="cursor-pointer text-sm font-medium text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200">
            {t('location_details')}
          </summary>
          <div className="mt-3 space-y-4">
            <Input
              label={t('location')}
              placeholder="e.g., Building A, Floor 3"
              error={fieldError('location')}
              {...register('location')}
            />
            <div className="grid grid-cols-2 gap-4">
              <Input
                label={t('latitude')}
                type="number"
                step="any"
                placeholder="53.9023"
                error={fieldError('latitude')}
                {...register('latitude', { valueAsNumber: true })}
              />
              <Input
                label={t('longitude')}
                type="number"
                step="any"
                placeholder="27.5618"
                error={fieldError('longitude')}
                {...register('longitude', { valueAsNumber: true })}
              />
            </div>
          </div>
        </details>

        {/* ═══ Actions ══════════════════════════════════════════════════ */}
        <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
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
