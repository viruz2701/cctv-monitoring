// ═══════════════════════════════════════════════════════════════════════
// Logging Settings — Audit Trail Configuration (ISO 27001 A.12.4)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { FileText } from '../components/ui/Icons';
import { useTranslation } from 'react-i18next';
import { Card, CardHeader, CardBody, Input, Select } from '../../components/ui';

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LoggingConfig {
  level: LogLevel;
  retention_days: number;
  port?: number;
}

interface Props {
  logging: LoggingConfig;
  onChange: (logging: LoggingConfig) => void;
}

const LOG_LEVELS: { value: LogLevel; label: string }[] = [
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
];

const RETENTION_OPTIONS = [
  { value: 7, label: '7 days' },
  { value: 30, label: '30 days' },
  { value: 90, label: '90 days' },
];

export function LoggingSettings({ logging, onChange }: Props) {
  const { t } = useTranslation();

  const handleChange = (field: keyof LoggingConfig, value: string | number) => {
    onChange({ ...logging, [field]: value });
  };

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <FileText className="w-5 h-5 text-slate-500 dark:text-slate-400" />
        <span className="text-lg font-semibold text-slate-900 dark:text-white">
          {t('logging') || 'Logging'}
        </span>
      </CardHeader>
      <CardBody className="space-y-4">
        <Select
          label={t('log_level') || 'Log Level'}
          value={logging.level}
          onChange={(e) => handleChange('level', e.target.value)}
          options={LOG_LEVELS.map((l) => ({ value: l.value, label: l.label }))}
        />

        <Select
          label={t('retention_days') || 'Retention Days'}
          value={logging.retention_days}
          onChange={(e) => handleChange('retention_days', Number(e.target.value))}
          options={RETENTION_OPTIONS.map((r) => ({ value: String(r.value), label: r.label }))}
        />

        {logging.level === 'debug' && (
          <Input
            label={t('log_server_port') || 'Log Server Port'}
            type="number"
            value={logging.port ?? 9514}
            onChange={(e) => handleChange('port', Number(e.target.value))}
            min={1024}
            max={65535}
          />
        )}
      </CardBody>
    </Card>
  );
}
