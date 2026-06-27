import { z } from 'zod';

// ─── Enums ──────────────────────────────────────────────────────────────────
export const CONNECTION_TYPES = [
  'ip',
  'p2p',
  'snmp',
  'syslog',
  'alarm',
  'gb28181',
  'onvif',
] as const;

export const DEVICE_TYPES = ['camera', 'nvr', 'dvr', 'switch'] as const;

export const SNMP_VERSIONS = ['v1', 'v2c', 'v3'] as const;

export const ALARM_PROTOCOLS = ['http', 'sip', 'xml'] as const;

export const P2P_BRANDS = [
  'hikvision',
  'dahua',
  'reolink',
  'xiongmai',
  'ezviz',
] as const;

// ─── Zod Enums ──────────────────────────────────────────────────────────────
export const connectionTypeSchema = z.enum(CONNECTION_TYPES, {
  message: 'validation.invalid_connection_type',
});

export const deviceTypeSchema = z.enum(DEVICE_TYPES, {
  message: 'validation.invalid_device_type',
});

// ─── Reusable validators ────────────────────────────────────────────────────
const ipv4Regex = /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)$/;

const ipAddressField = z
  .string()
  .regex(ipv4Regex, 'validation.invalid_ip')
  .optional();

const portField = z.coerce
  .number()
  .int('validation.port_must_be_integer')
  .min(1, 'validation.port_min')
  .max(65535, 'validation.port_max')
  .optional();

// ─── Main AddDevice Schema ─────────────────────────────────────────────────
export const addDeviceSchema = z
  .object({
    // ── Core required ────────────────────────────────────────────────────
    name: z
      .string()
      .min(2, 'validation.name_required')
      .max(255, 'validation.name_too_long'),

    siteId: z
      .string()
      .min(1, 'validation.site_required'),

    connectionType: connectionTypeSchema,

    deviceType: deviceTypeSchema.optional(),

    model: z
      .string()
      .max(100, 'validation.model_too_long')
      .optional(),

    // ── Location ─────────────────────────────────────────────────────────
    location: z
      .string()
      .max(500, 'validation.location_too_long')
      .optional(),

    latitude: z.coerce
      .number()
      .min(-90, 'validation.latitude_range')
      .max(90, 'validation.latitude_range')
      .optional(),

    longitude: z.coerce
      .number()
      .min(-180, 'validation.longitude_range')
      .max(180, 'validation.longitude_range')
      .optional(),

    // ── IP connection ────────────────────────────────────────────────────
    ipAddress: ipAddressField,
    port: portField,

    // ── P2P connection ───────────────────────────────────────────────────
    p2pBrand: z
      .enum(P2P_BRANDS)
      .optional(),

    p2pSerial: z
      .string()
      .optional(),

    p2pSecurityCode: z
      .string()
      .optional(),

    p2pCloudUser: z
      .string()
      .optional(),

    p2pCloudPass: z
      .string()
      .optional(),

    // ── ONVIF connection ─────────────────────────────────────────────────
    onvifUsername: z
      .string()
      .optional(),

    onvifPassword: z
      .string()
      .optional(),

    // ── SNMP connection ──────────────────────────────────────────────────
    snmpCommunity: z
      .string()
      .optional(),

    snmpVersion: z
      .enum(SNMP_VERSIONS)
      .optional(),

    // ── Syslog connection ────────────────────────────────────────────────
    syslogPort: portField,

    // ── Alarm connection ─────────────────────────────────────────────────
    alarmProtocol: z
      .enum(ALARM_PROTOCOLS)
      .optional(),
  })
  .superRefine((data, ctx) => {
    // ═══ Conditional: IP connection ═══════════════════════════════════════
    if (data.connectionType === 'ip' || data.connectionType === 'onvif') {
      if (!data.ipAddress || data.ipAddress.trim() === '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['ipAddress'],
          message: 'validation.ip_required',
        });
      }
    }

    // ═══ Conditional: P2P connection ═════════════════════════════════════
    if (data.connectionType === 'p2p') {
      if (!data.p2pSerial || data.p2pSerial.trim() === '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['p2pSerial'],
          message: 'validation.serial_required',
        });
      }
    }

    // ═══ Conditional: ONVIF credentials ══════════════════════════════════
    if (data.connectionType === 'onvif') {
      if (!data.onvifUsername || data.onvifUsername.trim() === '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['onvifUsername'],
          message: 'validation.onvif_username_required',
        });
      }
      if (!data.onvifPassword || data.onvifPassword.trim() === '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['onvifPassword'],
          message: 'validation.onvif_password_required',
        });
      }
    }

    // ═══ Conditional: Syslog port ════════════════════════════════════════
    if (data.connectionType === 'syslog' && data.syslogPort !== undefined) {
      const port = Number(data.syslogPort);
      if (!Number.isInteger(port) || port < 1 || port > 65535) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['syslogPort'],
          message: 'validation.invalid_port',
        });
      }
    }
  });

// ─── Types ──────────────────────────────────────────────────────────────────
export type AddDeviceFormData = z.infer<typeof addDeviceSchema>;

// ─── Default values for react-hook-form ────────────────────────────────────
export const ADD_DEVICE_DEFAULTS: AddDeviceFormData = {
  name: '',
  siteId: '',
  connectionType: 'ip',
  deviceType: undefined,
  model: '',
  location: '',
  latitude: undefined,
  longitude: undefined,
  ipAddress: '',
  port: undefined,
  p2pBrand: 'hikvision',
  p2pSerial: '',
  p2pSecurityCode: '',
  p2pCloudUser: '',
  p2pCloudPass: '',
  onvifUsername: '',
  onvifPassword: '',
  snmpCommunity: 'public',
  snmpVersion: 'v2c',
  syslogPort: undefined,
  alarmProtocol: 'http',
};
