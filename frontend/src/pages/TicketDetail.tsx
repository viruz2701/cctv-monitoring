import { generateUUID } from '../utils/uuid';
import React, { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardBody, Button, Badge, TicketStatusBadge, PriorityBadge, Textarea, ConfirmModal } from '../components/ui';
import { Clock, MapPin, HardDrive, User, Send, Trash2, AlertTriangle } from 'lucide-react';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useAuth } from '../hooks/useAuth';
import type { Ticket, TicketStatus, TicketComment as UITicketComment } from '../types';
import type { TicketComment } from '../services/api';
import { useTickets, useUpdateTicket, useDeleteTicket } from '../hooks/useApiQuery';
import { useTranslation } from 'react-i18next';
import { api, Alarm } from '../services/api';
import { Breadcrumbs } from '../components/ui/Breadcrumbs';

export function TicketDetail() {
    const { t } = useTranslation();
    const { ticketId } = useParams();
    const navigate = useNavigate();
    const { user } = useAuth();
    const { data: apiTickets = [] } = useTickets();
    const updateTicketMut = useUpdateTicket();
    const deleteTicketMut = useDeleteTicket();

    // API→UI mapping (migrated from TicketsContext)
    const tickets = useMemo(() => apiTickets.map(t => ({
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as Ticket['priority']) || 'medium',
        status: (t.status as Ticket['status']) || 'open',
        assignee: t.assignee || '',
        createdAt: t.created_at,
        updatedAt: t.updated_at,
        comments: (t.comments || []).map((c: TicketComment) => ({
            id: c.id,
            ticketId: c.ticket_id,
            userId: c.user_id || '',
            userName: c.user_name || '',
            content: c.content,
            createdAt: c.created_at,
        })),
    })), [apiTickets]);

    const ticket = tickets.find(t => t.id === ticketId);

    const [newComment, setNewComment] = useState('');
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [alarms, setAlarms] = useState<Alarm[]>([]);
    const [alarmsLoading, setAlarmsLoading] = useState(false);

    useEffect(() => {
      if (ticket?.deviceId) {
        setAlarmsLoading(true);
        api.getAlarms(ticket.deviceId)
          .then(setAlarms)
          .catch(() => {})
          .finally(() => setAlarmsLoading(false));
      }
    }, [ticket?.deviceId]);

    if (!ticket) {
        return (
            <div className="flex flex-col items-center justify-center h-96">
                <h2 className="text-xl font-bold text-slate-900 dark:text-white">{t('ticket_not_found')}</h2>
                <Button variant="outline" className="mt-4" onClick={() => navigate('/tickets')}>
                    {t('return_to_tickets')}
                </Button>
            </div>
        );
    }

    const handleStatusChange = (newStatus: TicketStatus) => {
        updateTicketMut.mutateAsync({ id: ticket.id, updates: { status: newStatus } });
    };

    const handleAddComment = () => {
        if (!newComment.trim() || !user) return;

        const comment: UITicketComment = {
            id: `cm-${generateUUID()}`,
            ticketId: ticket.id,
            userId: user.id,
            userName: user.name || user.username || t('anonymous'),
            content: newComment,
            createdAt: new Date().toISOString(),
        };

        (async () => {
            const { api } = await import('../services/api');
            await api.addTicketComment(ticket.id, comment.content);
        })();
        setNewComment('');
    };

    const handleDeleteTicket = () => {
        setShowDeleteConfirm(true);
    };

    const confirmDeleteTicket = () => {
        deleteTicketMut.mutateAsync(ticket.id);
        navigate('/tickets');
    };

    return (
        <div className="max-w-4xl mx-auto space-y-6">
            <Breadcrumbs
                items={[
                    { label: 'tickets', href: '/tickets' },
                    { label: ticket.title, href: undefined },
                ]}
                className="mb-2"
            />

            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
                        {ticket.title}
                    </h1>
                    <div className="flex items-center gap-3 text-sm text-slate-500 dark:text-slate-400">
                        <span className="font-mono bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                            {ticket.id}
                        </span>
                        <span>•</span>
                        <div className="flex items-center gap-1">
                            <Clock className="w-3.5 h-3.5" />
                            {new Date(ticket.createdAt).toLocaleString()}
                        </div>
                    </div>
                </div>
                <div className="flex gap-2">
                    <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
                        {(ticket.status === 'open' || ticket.status === 'in_progress') && (
                            <Button
                                variant="outline"
                                icon={<Clock className="w-4 h-4" />}
                                onClick={() => handleStatusChange('pending')}
                            >
                                {t('mark_pending')}
                            </Button>
                        )}
                        {ticket.status !== 'closed' && (
                            <Button
                                variant={ticket.status === 'resolved' ? 'outline' : 'primary'}
                                onClick={() => handleStatusChange('resolved')}
                                disabled={ticket.status === 'resolved'}
                            >
                                {t('mark_resolved')}
                            </Button>
                        )}
                        {ticket.status === 'resolved' && (
                            <Button onClick={() => handleStatusChange('closed')}>
                                {t('close_ticket')}
                            </Button>
                        )}

                        <PermissionGuard requiredRole={['admin', 'manager']}>
                            <Button
                                variant="outline"
                                className="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-900/30 dark:hover:bg-red-900/20"
                                onClick={handleDeleteTicket}
                                icon={<Trash2 className="w-4 h-4" />}
                            >
                                {t('delete')}
                            </Button>
                        </PermissionGuard>
                    </PermissionGuard>
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div className="lg:col-span-2 space-y-6">
                    <Card>
                        <CardBody className="space-y-4">
                            <h3 className="font-semibold text-slate-900 dark:text-white">{t('description')}</h3>
                            <p className="text-slate-700 dark:text-slate-300 leading-relaxed">
                                {ticket.description}
                            </p>
                        </CardBody>
                    </Card>

                    <div className="space-y-4">
                        <h3 className="font-semibold text-slate-900 dark:text-white">{t('activity_comments')}</h3>

                        <div className="space-y-4">
                            {ticket.comments?.map(comment => (
                                <div key={comment.id} className="flex gap-3">
                                    <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-xs font-bold text-blue-700 dark:text-blue-300 flex-shrink-0">
                                        {comment.userName.charAt(0)}
                                    </div>
                                    <div className="flex-1 bg-white dark:bg-slate-800 p-4 rounded-lg border border-slate-200 dark:border-slate-700 shadow-sm">
                                        <div className="flex justify-between items-start mb-1">
                                            <span className="font-medium text-slate-900 dark:text-white text-sm">
                                                {comment.userName}
                                            </span>
                                            <span className="text-xs text-slate-500">
                                                {new Date(comment.createdAt).toLocaleString()}
                                            </span>
                                        </div>
                                        <p className="text-slate-700 dark:text-slate-300 text-sm">
                                            {comment.content}
                                        </p>
                                    </div>
                                </div>
                            ))}
                            {(!ticket.comments || ticket.comments.length === 0) && (
                                <p className="text-center text-slate-500 py-4 italic">{t('no_comments')}</p>
                            )}
                        </div>

                        <PermissionGuard requireManageTickets>
                            <div className="flex gap-3 mt-6">
                                <div className="flex-1">
                                    <Textarea
                                        placeholder={t('add_comment')}
                                        value={newComment}
                                        onChange={(e) => setNewComment(e.target.value)}
                                        rows={3}
                                    />
                                </div>
                                <Button
                                    className="self-end"
                                    icon={<Send className="w-4 h-4" />}
                                    onClick={handleAddComment}
                                    disabled={!newComment.trim()}
                                >
                                    {t('reply')}
                                </Button>
                            </div>
                        </PermissionGuard>
                    </div>
                </div>

                <div className="space-y-6">
                    <Card>
                        <CardBody className="space-y-4">
                            <h3 className="font-semibold text-slate-900 dark:text-white mb-2">{t('details')}</h3>

                            <div className="space-y-3">
                                <div className="flex justify-between items-center">
                                    <span className="text-sm text-slate-500">{t('status')}</span>
                                    <TicketStatusBadge status={ticket.status} />
                                </div>
                                <div className="flex justify-between items-center">
                                    <span className="text-sm text-slate-500">{t('priority')}</span>
                                    <PriorityBadge priority={ticket.priority} />
                                </div>
                            </div>

                            <hr className="my-2 border-slate-100 dark:border-slate-800" />

                            <div className="space-y-3">
                                <div className="flex items-start gap-3">
                                    <MapPin className="w-4 h-4 text-slate-400 mt-0.5" />
                                    <div>
                                        <p className="text-xs text-slate-500">{t('site')}</p>
                                        <p className="text-sm font-medium text-slate-900 dark:text-white">{ticket.siteName}</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-3">
                                    <HardDrive className="w-4 h-4 text-slate-400 mt-0.5" />
                                    <div>
                                        <p className="text-xs text-slate-500">{t('device')}</p>
                                        <p className="text-sm font-medium text-slate-900 dark:text-white">{ticket.deviceName}</p>
                                    </div>
                                </div>
                                <div className="flex items-start gap-3">
                                    <User className="w-4 h-4 text-slate-400 mt-0.5" />
                                    <div>
                                        <p className="text-xs text-slate-500">{t('assignee')}</p>
                                        <p className="text-sm font-medium text-slate-900 dark:text-white">{ticket.assignee}</p>
                                    </div>
                                </div>
                            </div>
                        </CardBody>
                    </Card>

                    {/* ALARMS SECTION */}
                    {ticket?.deviceId && (
                      <Card>
                        <CardBody>
                          <div className="flex items-center gap-2 mb-4">
                            <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                            <h3 className="font-semibold text-slate-900 dark:text-white">{t('device_alarms') || 'Device Alarms'}</h3>
                          </div>
                          {alarmsLoading ? (
                            <div className="flex items-center justify-center py-4">
                              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-amber-600"></div>
                              <span className="ml-2 text-sm text-slate-500">{t('loading')}</span>
                            </div>
                          ) : alarms.length === 0 ? (
                            <p className="text-sm text-slate-500 dark:text-slate-400 text-center py-4">
                              {t('no_alarms') || 'No alarms'}
                            </p>
                          ) : (
                            <div className="space-y-2 max-h-60 overflow-y-auto">
                              {alarms.map((alarm, idx) => (
                                <div key={alarm.timestamp + alarm.device_id + idx} className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800 rounded-lg text-sm">
                                  <AlertTriangle className={`w-4 h-4 flex-shrink-0 ${alarm.priority >= 3 ? 'text-red-500' : alarm.priority >= 2 ? 'text-amber-500' : 'text-blue-500'}`} />
                                  <div className="flex-1 min-w-0">
                                    <p className="font-medium text-slate-900 dark:text-white truncate">{alarm.description}</p>
                                    <p className="text-xs text-slate-500 dark:text-slate-400">{new Date(alarm.timestamp).toLocaleString()}</p>
                                  </div>
                                  <Badge variant={alarm.priority >= 3 ? 'danger' : alarm.priority >= 2 ? 'warning' : 'info'}>
                                    P{alarm.priority}
                                  </Badge>
                                </div>
                              ))}
                            </div>
                          )}
                        </CardBody>
                      </Card>
                    )}
                </div>
            </div>

            <ConfirmModal
                isOpen={showDeleteConfirm}
                onClose={() => setShowDeleteConfirm(false)}
                onConfirm={confirmDeleteTicket}
                title={t('delete_ticket')}
                message={t('delete_ticket_confirm')}
                confirmText={t('delete')}
                cancelText={t('cancel')}
                variant="danger"
            />
        </div>
    );
}