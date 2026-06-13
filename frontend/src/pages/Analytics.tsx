import React, { useState, useEffect } from 'react';
import { Card, Table, Badge } from '../components/ui';
import { api, Prediction } from '../services/api';
import { useAuth } from '../hooks/useAuth';
import { useTranslation } from 'react-i18next';

export function Analytics() {
    const { t } = useTranslation();
    const { token } = useAuth();
    const [predictions, setPredictions] = useState<Prediction[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    useEffect(() => {
        if (!token) return;
        api.getPredictions()
            .then(data => setPredictions(data))
            .catch(err => setError(err.message))
            .finally(() => setLoading(false));
    }, [token]);

    if (loading) return <div className="p-8 text-center">{t('loading')}</div>;
    if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('analytics_predictions')}</h1>
            <Card>
                <Table
                    data={predictions}
                    columns={[
                        { header: t('device_id'), key: 'device_id' },
                        { header: t('failure_probability'), key: 'failure_probability', render: (p: Prediction) => <Badge variant={p.failure_probability > 70 ? 'danger' : p.failure_probability > 30 ? 'warning' : 'success'}>{p.failure_probability}%</Badge> },
                        { header: t('explanation'), key: 'explanation' },
                    ]}
                    keyExtractor={(p) => p.device_id + p.prediction_date}
                    emptyMessage={t('no_predictions')}
                />
            </Card>
        </div>
    );
}