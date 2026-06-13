import { generateUUID } from '../utils/uuid';
import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardBody, Button, Badge, TicketStatusBadge, PriorityBadge, Textarea, ConfirmModal } from '../components/ui';
import { ArrowLeft, Clock, MapPin, HardDrive, User, Send, Trash2 } from 'lucide-react';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useAuth } from '../hooks/useAuth';
import type { Ticket, TicketStatus, TicketComment } from '../types';
import { useTickets } from '../context/DataContext';
import { useTranslation } from 'react-i18next';

export function TicketDetail() {
    const { t } = useTranslation();
    const { ticketId } = useParams();
    const navigate = useNavigate();
    const { user } = useAuth();
    const { tickets, updateTicket, addTicketComment, deleteTicket } = useTickets();

    const ticket = tickets.find(t => t.id === ticketId);

    const [newComment, setNewComment] = useState('');
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

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
        updateTicket(ticket.id, { status: newStatus });
    };

    const handleAddComment = () => {
        if (!newComment.trim() || !user) return;

        const comment: TicketComment = {
            id: `cm-${generateUUID()}`,
            ticketId: ticket.id,
            userId: user.id,
            userName: user.name || user.username || t('anonymous'),
            content: newComment,
            createdAt: new Date().toISOString(),
        };

        addTicketComment(ticket.id, comment);
        setNewComment('');
    };

    const handleDeleteTicket = () => {
        setShowDeleteConfirm(true);
    };

    const confirmDeleteTicket = () => {
        deleteTicket(ticket.id);
        navigate('/tickets');
    };

    return (
        <div className="max-w-4xl mx-auto space-y-6">
            <Button variant="ghost" onClick={() => navigate('/tickets')} className="pl-0 gap-2">
                <ArrowLeft className="w-4 h-4" /> {t('back_to_tickets')}
            </Button>

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