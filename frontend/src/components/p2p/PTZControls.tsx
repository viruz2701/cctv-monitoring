import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, useToast } from '../ui';
import { p2pApi } from '../../services/p2pApi';
import { PTZCommand } from '../../types';
import { MoveUp, MoveDown, MoveLeft, MoveRight, ZoomIn, ZoomOut } from '../ui/Icons';

interface PTZControlsProps {
    deviceId: string;
    disabled?: boolean;
}

export const PTZControls: React.FC<PTZControlsProps> = ({ deviceId, disabled = false }) => {
    const { t } = useTranslation();
    const toast = useToast();
    const [loading, setLoading] = useState<string | null>(null);

    const sendCommand = async (command: PTZCommand['command']) => {
        if (disabled || loading) return;
        setLoading(command);
        try {
            await p2pApi.sendCommand(deviceId, { command });
        } catch (err) {
            toast.error(t('ptz_command_failed'));
        } finally {
            setLoading(null);
        }
    };

    const buttonClass =
        "p-3 bg-slate-100 dark:bg-slate-700 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors disabled:opacity-50";

    return (
        <div className="grid grid-cols-3 gap-2 max-w-[200px]">
            <div></div>
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('up')}
                disabled={disabled || loading === 'up'}
                icon={<MoveUp className="w-5 h-5" />}
            />
            <div></div>
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('left')}
                disabled={disabled || loading === 'left'}
                icon={<MoveLeft className="w-5 h-5" />}
            />
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('down')}
                disabled={disabled || loading === 'down'}
                icon={<MoveDown className="w-5 h-5" />}
            />
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('right')}
                disabled={disabled || loading === 'right'}
                icon={<MoveRight className="w-5 h-5" />}
            />
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('zoom_in')}
                disabled={disabled || loading === 'zoom_in'}
                icon={<ZoomIn className="w-5 h-5" />}
            />
            <Button
                variant="ghost"
                className={buttonClass}
                onClick={() => sendCommand('zoom_out')}
                disabled={disabled || loading === 'zoom_out'}
                icon={<ZoomOut className="w-5 h-5" />}
            />
            <div></div>
        </div>
    );
};