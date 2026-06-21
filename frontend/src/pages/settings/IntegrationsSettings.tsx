import React from 'react';
import { Globe } from 'lucide-react';
import { Card, CardHeader, CardBody } from '../../components/ui';
import { useTranslation } from 'react-i18next';

interface Props {
  children?: React.ReactNode;
}

export const IntegrationsSettings: React.FC<Props> = ({ children }) => {
  const { t } = useTranslation();

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Globe className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
          <div>
            <span>Atlas CMMS</span>
            <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">
              External CMMS integration with OAuth2 and offline fallback queue
            </p>
          </div>
        </CardHeader>
        <CardBody>
          {children}
        </CardBody>
      </Card>
    </div>
  );
};