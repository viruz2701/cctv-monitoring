import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import {
  Card,
  CardHeader,
  CardBody,
  Badge,
  Button,
  Modal,
  Input,
  Table,
  EmptyState,
  SkeletonTable,
  useToast,
} from '../components/ui';
import {
  Play,
  RotateCcw,
  RefreshCw,
  Search,
  Filter,
  X,
  List,
  Database,
  Server,
  HardDrive,
  Clock,
  ChevronDown,
  Loader2,
  AlertTriangle,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface StreamInfo {
  name: string;
  description?: string;
  subjects: string;
  msg_count: number;
  byte_size: number;
  max_age: string;
  storage: string;
  retention: string;
}

interface MessageInfo {
  seq: number;
  subject: string;
  data: unknown;
  timestamp: string;
  stream_name: string;
}

interface StreamListResponse {
  streams: StreamInfo[];
}

interface MessageListResponse {
  messages: MessageInfo[];
  stream: string;
  limit: number;
  offset: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

function getRetentionVariant(retention: string): 'info' | 'success' | 'warning' | 'neutral' {
  switch (retention) {
    case 'limits': return 'info';
    case 'interest': return 'success';
    case 'work_queue': return 'warning';
    default: return 'neutral';
  }
}

function getStorageVariant(storage: string): 'info' | 'primary' {
  return storage === 'file' ? 'info' : 'primary';
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function EventReplay() {
  const { t } = useTranslation();
  const toast = useToast();

  // State
  const [streams, setStreams] = useState<StreamInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selected stream
  const [selectedStream, setSelectedStream] = useState<string | null>(null);
  const [messages, setMessages] = useState<MessageInfo[]>([]);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [messagesOffset, setMessagesOffset] = useState(0);
  const [hasMoreMessages, setHasMoreMessages] = useState(true);

  // DLQ
  const [dlqMessages, setDlqMessages] = useState<MessageInfo[]>([]);
  const [dlqLoading, setDlqLoading] = useState(false);
  const [showDLQ, setShowDLQ] = useState(false);

  // Search
  const [searchSubject, setSearchSubject] = useState('');
  const [searchTenant, setSearchTenant] = useState('');

  // Replay modal
  const [replayModal, setReplayModal] = useState<{
    open: boolean;
    stream: string;
    seq: number;
    subject: string;
  }>({ open: false, stream: '', seq: 0, subject: '' });

  // Message detail modal
  const [detailModal, setDetailModal] = useState<{
    open: boolean;
    message: MessageInfo | null;
  }>({ open: false, message: null });

  const MESSAGE_LIMIT = 50;

  // ── Load streams ───────────────────────────────────────────────────

  const loadStreams = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await request<StreamListResponse>('/events/streams');
      setStreams(data.streams ?? []);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load streams';
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    loadStreams();
  }, [loadStreams]);

  // ── Load messages for a stream ─────────────────────────────────────

  const loadMessages = useCallback(async (stream: string, offset: number, _append = false) => {
    setMessagesLoading(true);
    try {
      const params = new URLSearchParams({
        limit: String(MESSAGE_LIMIT),
        offset: String(offset),
      });
      if (searchSubject) params.set('subject', searchSubject);
      if (searchTenant) params.set('tenant_id', searchTenant);

      const data = await request<MessageListResponse>(
        `/events/streams/${encodeURIComponent(stream)}/messages?${params}`,
      );

      if (_append) {
        setMessages(prev => [...prev, ...(data.messages ?? [])]);
      } else {
        setMessages(data.messages ?? []);
      }
      setHasMoreMessages((data.messages ?? []).length >= MESSAGE_LIMIT);
      setMessagesOffset(offset);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load messages';
      toast.error(msg);
    } finally {
      setMessagesLoading(false);
    }
  }, [searchSubject, searchTenant, toast]);

  const handleStreamSelect = useCallback((stream: string) => {
    setSelectedStream(stream);
    setMessages([]);
    setMessagesOffset(0);
    setHasMoreMessages(true);
    loadMessages(stream, 0);
  }, [loadMessages]);

  // ── Load DLQ ───────────────────────────────────────────────────────

  const loadDLQ = useCallback(async () => {
    setDlqLoading(true);
    try {
      const params = new URLSearchParams({ limit: String(MESSAGE_LIMIT), offset: '0' });
      if (searchSubject) params.set('subject', searchSubject);
      if (searchTenant) params.set('tenant_id', searchTenant);

      const data = await request<MessageListResponse>(
        `/events/dead-letters?${params}`,
      );
      setDlqMessages(data.messages ?? []);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load DLQ';
      toast.error(msg);
    } finally {
      setDlqLoading(false);
    }
  }, [searchSubject, searchTenant, toast]);

  const toggleDLQ = useCallback(() => {
    setShowDLQ(prev => {
      if (!prev) {
        loadDLQ();
      }
      return !prev;
    });
  }, [loadDLQ]);

  // ── Replay message ─────────────────────────────────────────────────

  const handleReplay = useCallback(async () => {
    const { stream, seq } = replayModal;
    try {
      await request(`/events/streams/${encodeURIComponent(stream)}/replay/${seq}`, {
        method: 'POST',
      });
      toast.success(`Message #${seq} replayed successfully`);
      setReplayModal({ open: false, stream: '', seq: 0, subject: '' });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Replay failed';
      toast.error(msg);
    }
  }, [replayModal, toast]);

  // ── Columns (P3-MICRO.2: мемоизированы для предотвращения ререндеров Table) ──
  const streamsColumns = useMemo(() => [
    {
      key: 'name' as const,
      header: t('events.stream_name', 'Stream'),
      render: (s: StreamInfo) => (
        <button
          onClick={() => handleStreamSelect(s.name)}
          className={`font-mono text-sm hover:underline ${
            selectedStream === s.name ? 'text-blue-600 dark:text-blue-400 font-semibold' : 'text-blue-500'
          }`}
        >
          {s.name}
        </button>
      ),
    },
    {
      key: 'subjects' as const,
      header: t('events.subjects', 'Subjects'),
      render: (s: StreamInfo) => (
        <span className="text-xs font-mono text-gray-500 dark:text-gray-400 truncate max-w-[200px] block">
          {s.subjects}
        </span>
      ),
    },
    {
      key: 'msg_count' as const,
      header: t('events.messages', 'Messages'),
      render: (s: StreamInfo) => (
        <span className="font-mono text-sm">{s.msg_count.toLocaleString()}</span>
      ),
    },
    {
      key: 'byte_size' as const,
      header: t('events.size', 'Size'),
      render: (s: StreamInfo) => (
        <span className="font-mono text-sm text-gray-600 dark:text-gray-400">
          {formatBytes(s.byte_size)}
        </span>
      ),
    },
    {
      key: 'retention' as const,
      header: t('events.retention', 'Retention'),
      render: (s: StreamInfo) => (
        <Badge variant={getRetentionVariant(s.retention)}>{s.retention}</Badge>
      ),
    },
    {
      key: 'storage' as const,
      header: t('events.storage', 'Storage'),
      render: (s: StreamInfo) => (
        <Badge variant={getStorageVariant(s.storage)}>{s.storage}</Badge>
      ),
    },
    {
      key: 'max_age' as const,
      header: t('events.max_age', 'Max Age'),
      render: (s: StreamInfo) => (
        <span className="text-sm text-gray-500">{s.max_age}</span>
      ),
    },
  ], [t, handleStreamSelect, selectedStream]);

  const messagesColumns = useMemo(() => [
    {
      key: 'seq' as const,
      header: '#',
      width: '60px',
      render: (m: MessageInfo) => (
        <span className="font-mono text-xs text-gray-500">#{m.seq}</span>
      ),
    },
    {
      key: 'subject' as const,
      header: t('events.subject', 'Subject'),
      render: (m: MessageInfo) => (
        <span className="font-mono text-xs text-blue-600 dark:text-blue-400">{m.subject}</span>
      ),
    },
    {
      key: 'timestamp' as const,
      header: t('events.timestamp', 'Timestamp'),
      render: (m: MessageInfo) => (
        <span className="text-xs text-gray-500">{formatTime(m.timestamp)}</span>
      ),
    },
    {
      key: 'data_preview' as const,
      header: t('events.data_preview', 'Data Preview'),
      render: (m: MessageInfo) => {
        const preview = JSON.stringify(m.data).slice(0, 80);
        return (
          <span className="text-xs font-mono text-gray-600 dark:text-gray-400 truncate max-w-[300px] block">
            {preview}{JSON.stringify(m.data).length > 80 ? '…' : ''}
          </span>
        );
      },
    },
    {
      key: 'actions' as const,
      header: t('events.actions', 'Actions'),
      width: '120px',
      render: (m: MessageInfo) => (
        <div className="flex gap-1">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setDetailModal({ open: true, message: m })}
            title={t('events.view_details', 'View Details')}
          >
            <Search className="w-3.5 h-3.5" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() =>
              setReplayModal({
                open: true,
                stream: m.stream_name,
                seq: m.seq,
                subject: m.subject,
              })
            }
            title={t('events.replay', 'Replay')}
          >
            <Play className="w-3.5 h-3.5 text-green-500" />
          </Button>
        </div>
      ),
    },
  ], [t]);

  // ── Render ─────────────────────────────────────────────────────────

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <RotateCcw className="w-6 h-6" />
            {t('events.page_title', 'Event Replay')}
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {t('events.page_subtitle', 'Browse and replay NATS JetStream events')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={toggleDLQ}>
            <AlertTriangle className="w-4 h-4 mr-1" />
            {t('events.dead_letters', 'Dead Letters')}
            {dlqMessages.length > 0 && (
              <Badge variant="danger" className="ml-1">{dlqMessages.length}</Badge>
            )}
          </Button>
          <Button variant="outline" size="sm" onClick={loadStreams}>
            <RefreshCw className="w-4 h-4 mr-1" />
            {t('common.refresh', 'Refresh')}
          </Button>
        </div>
      </div>

      {/* DLQ Panel */}
      {showDLQ && (
        <Card>
          <CardHeader className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-red-500" />
              <h2 className="text-lg font-semibold">
                {t('events.dead_letter_queue', 'Dead Letter Queue')}
              </h2>
            </div>
            <Button variant="ghost" size="sm" onClick={() => setShowDLQ(false)}>
              <X className="w-4 h-4" />
            </Button>
          </CardHeader>
          <CardBody>
            {dlqLoading ? (
              <SkeletonTable rows={5} columns={4} />
            ) : dlqMessages.length === 0 ? (
              <EmptyState
                icon={<CheckCircleIcon className="w-12 h-12" />}
                title={t('events.no_dlq', 'No dead letters')}
                description={t('events.no_dlq_desc', 'All messages are being processed successfully.')}
              />
            ) : (
              <Table
                data={dlqMessages}
                columns={messagesColumns}
                keyExtractor={(m: MessageInfo) => `${m.stream_name}-${m.seq}`}
                emptyMessage={t('events.no_messages', 'No messages')}
              />
            )}
          </CardBody>
        </Card>
      )}

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Streams List */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <Database className="w-5 h-5" />
                  {t('events.streams', 'Streams')}
                </h2>
                {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              </div>
            </CardHeader>
            <CardBody>
              {loading ? (
                <SkeletonTable rows={5} columns={3} />
              ) : error ? (
                <div className="text-red-500 text-sm p-4 text-center">
                  <p>{error}</p>
                  <Button variant="outline" size="sm" className="mt-2" onClick={loadStreams}>
                    {t('common.retry', 'Retry')}
                  </Button>
                </div>
              ) : streams.length === 0 ? (
                <EmptyState
                  icon={<Server className="w-12 h-12" />}
                  title={t('events.no_streams', 'No streams found')}
                  description={t('events.no_streams_desc', 'No JetStream streams are configured.')}
                />
              ) : (
                <div className="space-y-1">
                  {streams.map((stream) => (
                    <button
                      key={stream.name}
                      onClick={() => handleStreamSelect(stream.name)}
                      className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                        selectedStream === stream.name
                          ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                          : 'hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300'
                      }`}
                    >
                      <div className="font-mono font-medium">{stream.name}</div>
                      <div className="text-xs text-gray-400 mt-0.5">
                        {stream.msg_count.toLocaleString()} msgs — {formatBytes(stream.byte_size)}
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </CardBody>
          </Card>
        </div>

        {/* Messages */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between flex-wrap gap-2">
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <List className="w-5 h-5" />
                  {selectedStream
                    ? `${t('events.messages_for', 'Messages')}: ${selectedStream}`
                    : t('events.select_stream', 'Select a stream')}
                </h2>
                {selectedStream && (
                  <div className="flex items-center gap-2">
                    <Input
                      placeholder={t('events.filter_subject', 'Filter subject\u2026')}
                      value={searchSubject}
                      onChange={(e) => setSearchSubject(e.target.value)}
                      className="w-40 text-xs"
                    />
                    <Input
                      placeholder={t('events.filter_tenant', 'Tenant ID\u2026')}
                      value={searchTenant}
                      onChange={(e) => setSearchTenant(e.target.value)}
                      className="w-32 text-xs"
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => loadMessages(selectedStream, 0)}
                    >
                      <Filter className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                )}
              </div>
            </CardHeader>
            <CardBody>
              {!selectedStream ? (
                <EmptyState
                  icon={<HardDrive className="w-12 h-12" />}
                  title={t('events.select_stream_hint', 'Select a stream')}
                  description={t('events.select_stream_desc', 'Choose a stream from the left panel to view its messages.')}
                />
              ) : messagesLoading && messages.length === 0 ? (
                <SkeletonTable rows={8} columns={5} />
              ) : messages.length === 0 ? (
                <EmptyState
                  icon={<List className="w-12 h-12" />}
                  title={t('events.no_messages', 'No messages')}
                  description={t('events.no_messages_desc', 'This stream has no messages matching the filters.')}
                />
              ) : (
                <>
                  <Table
                    data={messages}
                    columns={messagesColumns}
                    keyExtractor={(m: MessageInfo) => `${m.stream_name}-${m.seq}`}
                    emptyMessage={t('events.no_messages', 'No messages')}
                  />
                  <div className="flex items-center justify-between mt-4">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={messagesOffset === 0}
                      onClick={() => {
                        const newOffset = Math.max(0, messagesOffset - MESSAGE_LIMIT);
                        loadMessages(selectedStream, newOffset);
                      }}
                    >
                      {t('common.previous', 'Previous')}
                    </Button>
                    <span className="text-xs text-gray-500">
                      {t('events.showing_offset', 'Showing offset {{offset}}', { offset: messagesOffset })}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!hasMoreMessages}
                      onClick={() => loadMessages(selectedStream, messagesOffset + MESSAGE_LIMIT, true)}
                    >
                      {messagesLoading ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin mr-1" />
                      ) : null}
                      {t('common.load_more', 'Load More')}
                    </Button>
                  </div>
                </>
              )}
            </CardBody>
          </Card>
        </div>
      </div>

      {/* ── Replay Confirmation Modal ────────────────────────────────── */}
      <Modal
        isOpen={replayModal.open}
        onClose={() => setReplayModal({ open: false, stream: '', seq: 0, subject: '' })}
        title={t('events.confirm_replay', 'Replay Message')}
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {t('events.replay_confirm_text', 'This will re-publish the message to its original subject.')}
          </p>
          <div className="bg-gray-50 dark:bg-gray-800 rounded-md p-3 space-y-1 text-sm font-mono">
            <div>
              <span className="text-gray-500">{t('events.stream', 'Stream')}: </span>
              <span className="font-semibold">{replayModal.stream}</span>
            </div>
            <div>
              <span className="text-gray-500">{t('events.sequence', 'Sequence')}: </span>
              <span className="font-semibold">#{replayModal.seq}</span>
            </div>
            <div>
              <span className="text-gray-500">{t('events.subject', 'Subject')}: </span>
              <span className="text-blue-600 dark:text-blue-400">{replayModal.subject}</span>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              onClick={() => setReplayModal({ open: false, stream: '', seq: 0, subject: '' })}
            >
              {t('common.cancel', 'Cancel')}
            </Button>
            <Button onClick={handleReplay}>
              <Play className="w-4 h-4 mr-1" />
              {t('events.replay', 'Replay')}
            </Button>
          </div>
        </div>
      </Modal>

      {/* ── Message Detail Modal ─────────────────────────────────────── */}
      <Modal
        isOpen={detailModal.open}
        onClose={() => setDetailModal({ open: false, message: null })}
        title={t('events.message_detail', 'Message Detail')}
      >
        {detailModal.message && (
          <div className="space-y-4">
            <div className="bg-gray-50 dark:bg-gray-800 rounded-md p-3 space-y-1 text-sm font-mono">
              <div>
                <span className="text-gray-500">{t('events.stream', 'Stream')}: </span>
                <span className="font-semibold">{detailModal.message.stream_name}</span>
              </div>
              <div>
                <span className="text-gray-500">{t('events.sequence', 'Sequence')}: </span>
                <span className="font-semibold">#{detailModal.message.seq}</span>
              </div>
              <div>
                <span className="text-gray-500">{t('events.subject', 'Subject')}: </span>
                <span className="text-blue-600 dark:text-blue-400">{detailModal.message.subject}</span>
              </div>
              <div>
                <span className="text-gray-500">{t('events.timestamp', 'Timestamp')}: </span>
                <span>{formatTime(detailModal.message.timestamp)}</span>
              </div>
            </div>
            <div>
              <h4 className="text-sm font-medium mb-2">{t('events.payload', 'Payload')}</h4>
              <pre className="bg-gray-900 dark:bg-gray-950 text-green-400 rounded-md p-4 text-xs overflow-auto max-h-96">
                {JSON.stringify(detailModal.message.data, null, 2)}
              </pre>
            </div>
            <div className="flex justify-end">
              <Button
                onClick={() =>
                  setReplayModal({
                    open: true,
                    stream: detailModal.message!.stream_name,
                    seq: detailModal.message!.seq,
                    subject: detailModal.message!.subject,
                  })
                }
              >
                <Play className="w-4 h-4 mr-1" />
                {t('events.replay', 'Replay')}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

// Fallback icon component for empty states
function CheckCircleIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  );
}
