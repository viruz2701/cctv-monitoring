import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Button, Input, Select, useToast } from '../ui';  // путь к ui компонентам
import { p2pApi } from '../../services/p2pApi';
import { P2PRegistrationForm as P2PForm } from '../../types';

interface Props {
    isOpen: boolean;
    onClose: () => void;
    onSuccess?: () => void;
}

export const P2PRegistrationForm: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
    const { t } = useTranslation();
    const toast = useToast();
    const [loading, setLoading] = useState(false);
    const [form, setForm] = useState<P2PForm>({
        serial: '',
        brand: 'hikvision',
        securityCode: '',
        username: '',
        password: '',
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            await p2pApi.register(form);
            toast.success(t('p2p_device_registered'));
            onSuccess?.();
            onClose();
        } catch (err: any) {
            toast.error(err.response?.data?.message || t('p2p_registration_failed'));
        } finally {
            setLoading(false);
        }
    };

    const brandOptions = [
        { value: 'hikvision', label: 'Hikvision' },
        { value: 'dahua', label: 'Dahua' },
        { value: 'xiongmai', label: 'Xiongmai' },
        { value: 'reolink', label: 'Reolink' },
        { value: 'ezviz', label: 'EZVIZ' },
    ];

    const needsCredentials = ['dahua', 'reolink', 'ezviz'].includes(form.brand);

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={t('add_p2p_device')}>
            <form onSubmit={handleSubmit} className="space-y-4">
                <Select
                    label={t('brand')}
                    options={brandOptions}
                    value={form.brand}
                    onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
                        setForm({ ...form, brand: e.target.value })
                    }
                />
                <Input
                    label={t('serial_number')}
                    placeholder="e.g., 95270DSD7FFRVTAS7"
                    value={form.serial}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                        setForm({ ...form, serial: e.target.value })
                    }
                    required
                />
                <Input
                    label={t('security_code')}
                    type="password"
                    placeholder={t('security_code_placeholder')}
                    value={form.securityCode}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                        setForm({ ...form, securityCode: e.target.value })
                    }
                    required
                />
                {needsCredentials && (
                    <>
                        <Input
                            label={t('username')}
                            placeholder="admin"
                            value={form.username}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                                setForm({ ...form, username: e.target.value })
                            }
                        />
                        <Input
                            label={t('password')}
                            type="password"
                            placeholder={t('password')}
                            value={form.password}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                                setForm({ ...form, password: e.target.value })
                            }
                        />
                    </>
                )}
                <div className="flex justify-end gap-3 pt-4">
                    <Button type="button" variant="outline" onClick={onClose}>
                        {t('cancel')}
                    </Button>
                    <Button type="submit" loading={loading}>
                        {t('register')}
                    </Button>
                </div>
            </form>
        </Modal>
    );
};