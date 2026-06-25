import React from 'react';
import { useTranslation } from 'react-i18next';
import { Clock } from 'lucide-react';
import { Card, CardHeader, CardBody, Timeline } from '../ui';
import { TimelineEvent } from '../ui/Timeline';

interface WODetailTimelineProps {
  events: TimelineEvent[];
}

export const WODetailTimeline: React.FC<WODetailTimelineProps> = ({ events }) => {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <Clock className="w-5 h-5 text-slate-600 dark:text-slate-400" />
        <span>{t('workOrder.history') || 'История'}</span>
      </CardHeader>
      <CardBody>
        <Timeline events={events} />
      </CardBody>
    </Card>
  );
};
