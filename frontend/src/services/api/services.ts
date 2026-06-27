// ═══════════════════════════════════════════════════════════════════════
// Services & Settings API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface SyslogSettings {
  enabled: boolean;
  udp_port: number;
  tcp_port: number;
}

export interface FTPSettings {
  enabled: boolean;
  port: number;
  user: string;
  password: string;
  root_path: string;
}

export interface SNMPV1Config {
  enabled: boolean;
  port: number;
  community: string;
}

export interface SNMPV2cConfig {
  enabled: boolean;
  port: number;
  community: string;
}

export interface SNMPV3Config {
  enabled: boolean;
  port: number;
  user: string;
  auth_protocol: 'MD5' | 'SHA' | 'SHA256';
  auth_password: string;
  priv_protocol: 'DES' | 'AES' | 'AES192' | 'AES256';
  priv_password: string;
}

export interface SNMPSettings {
  enabled: boolean;
  port: number;
  community: string;
  version: 'v1' | 'v2c' | 'v3';
  user?: string;
  auth_protocol?: 'MD5' | 'SHA' | 'SHA256';
  auth_password?: string;
  priv_protocol?: 'DES' | 'AES' | 'AES192' | 'AES256';
  priv_password?: string;
  v1_config: SNMPV1Config;
  v2c_config: SNMPV2cConfig;
  v3_config: SNMPV3Config;
}

export interface HTTPSettings {
  enabled: boolean;
  port: number;
}

export interface DahuaSettings {
  enabled: boolean;
  ports: number[];
}

export interface HisiliconSettings {
  enabled: boolean;
  port: number;
}

export interface TVTSettings {
  enabled: boolean;
  port: number;
}

export interface SIPSettings {
  enabled: boolean;
  port: number;
  host: string;
}

export interface GB28181Settings {
  enabled: boolean;
  port: number;
  host: string;
  server_id: string;
  server_ip: string;
  realm: string;
  auth_enabled: boolean;
  auth_user: string;
  auth_password: string;
  auto_catalog: boolean;
  auto_device_info: boolean;
  keepalive_interval: number;
  keepalive_timeout: number;
  max_sub_channels: number;
  log_sip_messages: boolean;
}

export interface P2PHikvisionSettings {
  username: string;
  password: string;
}

export interface P2PDahuaSettings {
  python_path: string;
  script_path: string;
}

export interface P2PReolinkSettings {
  proxy_bin_path: string;
}

export interface P2PXiongmaiSettings {
  uuid: string;
  app_key: string;
  app_secret: string;
  endpoint: string;
  region: string;
  move_card: number;
}

export interface P2PEZVIZSettings {
  app_key: string;
  app_secret: string;
}

export interface P2PGatewaySettings {
  url: string;
  api_key: string;
  enabled?: boolean;
  hikvision: P2PHikvisionSettings;
  dahua: P2PDahuaSettings;
  reolink: P2PReolinkSettings;
  xiongmai: P2PXiongmaiSettings;
  ezviz: P2PEZVIZSettings;
}

export interface ServicesSettings {
  services_syslog: SyslogSettings;
  services_ftp: FTPSettings;
  services_snmp: SNMPSettings;
  services_http: HTTPSettings;
  services_dahua: DahuaSettings;
  services_hisilicon: HisiliconSettings;
  services_tvt: TVTSettings;
  services_gb28181: GB28181Settings;
  services_p2p_gateway: P2PGatewaySettings;
}

// ─── API Methods ────────────────────────────────────────────────────

export const servicesApi = {
  getSettings(): Promise<ServicesSettings> {
    return request<ServicesSettings>('/settings/services');
  },

  updateSettings(settings: Partial<ServicesSettings>): Promise<{ status: string; restarted: string[] }> {
    return request<{ status: string; restarted: string[] }>('/settings/services', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
  },

  getStatus(): Promise<{ services: Record<string, { status: string; port: number; message?: string }> }> {
    return request<{ services: Record<string, { status: string; port: number; message?: string }> }>('/settings/services/status');
  },
};
