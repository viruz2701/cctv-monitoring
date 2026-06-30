import React, { useState, useRef, useEffect, useCallback } from 'react';
import { chatApi, type ChatMessage, type Attachment } from '../../services/chatApi';
import { useAuth } from '../../hooks/useAuth';
import {
  Send,
  Upload,
  Heart,
  MessageCircle,
  X,
  Check,
  ChevronDown,
  Loader2,
} from '../ui/Icons';

const MAX_RECONNECT_ATTEMPTS = 5;
const MAX_MESSAGE_LENGTH = 10000;
const TYPING_TIMEOUT = 3000;

function getWsBaseUrl(): string {
  if (import.meta.env.VITE_WS_URL) {
    return import.meta.env.VITE_WS_URL.replace('http', 'ws');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
}

interface WOChatProps {
  woId: string;
  isOpen: boolean;
  onClose: () => void;
}

interface MentionUser {
  id: string;
  name: string;
}

const EMOJI_LIST = ['👍', '❤️', '😄', '🎉', '😮', '😢', '👀'];

export function WOChat({ woId, isOpen, onClose }: WOChatProps) {
  const { token, user } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [before, setBefore] = useState<string | undefined>();
  const [typingUsers, setTypingUsers] = useState<string[]>([]);
  const [showMentions, setShowMentions] = useState(false);
  const [mentionSearch, setMentionSearch] = useState('');
  const [mentionCursor, setMentionCursor] = useState(0);
  const [activeUsers, setActiveUsers] = useState<MentionUser[]>([]);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const reconnectAttempts = useRef(0);
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const isTypingRef = useRef(false);
  const initialLoadDone = useRef(false);

  // ── Load history ────────────────────────────────────────────────────────

  const loadHistory = useCallback(async () => {
    try {
      setLoading(true);
      const res = await chatApi.getHistory(woId, 50, before);
      setMessages((prev) => {
        const existingIds = new Set(prev.map((m) => m.id));
        const newMsgs = res.messages.filter((m) => !existingIds.has(m.id));
        return [...newMsgs, ...prev];
      });
      if (res.messages.length < 50) {
        setHasMore(false);
      }
      if (res.messages.length > 0) {
        setBefore(res.messages[0].id);
      }
    } catch (err) {
      console.error('Failed to load chat history', err);
    } finally {
      setLoading(false);
    }
  }, [woId, before]);

  useEffect(() => {
    if (isOpen && !initialLoadDone.current) {
      initialLoadDone.current = true;
      setMessages([]);
      setBefore(undefined);
      setHasMore(true);
      loadHistory();
    }
    if (!isOpen) {
      initialLoadDone.current = false;
    }
  }, [isOpen, woId, loadHistory]);

  // ── WebSocket connection ────────────────────────────────────────────────

  useEffect(() => {
    if (!isOpen || !token || !user) return;

    const wsUrl = `${getWsBaseUrl()}/ws/chat/${woId}?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      reconnectAttempts.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleWsMessage(data);
      } catch (err) {
        console.error('Failed to parse WS message', err);
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (reconnectAttempts.current < MAX_RECONNECT_ATTEMPTS) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        reconnectAttempts.current += 1;
        setTimeout(() => {}, delay);
      }
    };

    ws.onerror = () => {
      console.error('Chat WebSocket error');
    };

    wsRef.current = ws;

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [isOpen, woId, token, user]);

  // ── WebSocket message handler ───────────────────────────────────────────

  const handleWsMessage = useCallback(
    (data: { type: string; payload?: unknown }) => {
      switch (data.type) {
        case 'chat': {
          const msg = data.payload as ChatMessage;
          setMessages((prev) => {
            if (prev.some((m) => m.id === msg.id)) return prev;
            return [...prev, msg];
          });
          scrollToBottom();
          break;
        }
        case 'typing': {
          const payload = data.payload as { user_id: string; user_name: string };
          setTypingUsers((prev) => {
            if (prev.includes(payload.user_name)) return prev;
            return [...prev, payload.user_name];
          });
          setTimeout(() => {
            setTypingUsers((prev) => prev.filter((u) => u !== payload.user_name));
          }, TYPING_TIMEOUT);
          break;
        }
        case 'presence': {
          const payload = data.payload as {
            users: Array<{ user_id: string; user_name: string }>;
          };
          setActiveUsers(
            (payload.users || []).map((u) => ({
              id: u.user_id,
              name: u.user_name,
            })),
          );
          break;
        }
        case 'read_receipt': {
          const payload = data.payload as { message_id: string; user_id: string };
          setMessages((prev) =>
            prev.map((m) =>
              m.id === payload.message_id
                ? { ...m, read_by: [...(m.read_by || []), payload.user_id] }
                : m,
            ),
          );
          break;
        }
        case 'reaction': {
          const payload = data.payload as { message_id: string; user_id: string; emoji: string };
          setMessages((prev) =>
            prev.map((m) =>
              m.id === payload.message_id ? { ...m, reaction: payload.emoji } : m,
            ),
          );
          break;
        }
        case 'error': {
          console.error('Chat WS error:', data.payload);
          break;
        }
      }
    },
    [],
  );

  // ── Send message ────────────────────────────────────────────────────────

  const sendMessage = useCallback(async () => {
    const text = input.trim();
    if (!text || sending) return;

    const mentionRegex = /@(\w+)/g;
    const mentions: string[] = [];
    const mentionMatches = text.matchAll(mentionRegex);
    for (const match of mentionMatches) {
      const mentionedUser = activeUsers.find((u) =>
        u.name.toLowerCase().includes(match[1].toLowerCase()),
      );
      if (mentionedUser && !mentions.includes(mentionedUser.id)) {
        mentions.push(mentionedUser.id);
      }
    }

    try {
      setSending(true);
      await chatApi.sendMessage(woId, {
        text,
        message_type: 'text',
        mentions: mentions.length > 0 ? mentions : undefined,
      });
      setInput('');
      scrollToBottom();
    } catch (err) {
      console.error('Failed to send message', err);
    } finally {
      setSending(false);
    }
  }, [input, sending, woId, activeUsers]);

  // ── Typing indicator ────────────────────────────────────────────────────

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    if (value.length > MAX_MESSAGE_LENGTH) return;
    setInput(value);

    const cursorPos = e.target.selectionStart;
    const textBeforeCursor = value.slice(0, cursorPos);
    const atMatch = textBeforeCursor.match(/@(\w*)$/);
    if (atMatch) {
      setMentionSearch(atMatch[1]);
      setShowMentions(true);
      setMentionCursor(0);
    } else {
      setShowMentions(false);
    }

    if (!isTypingRef.current && wsRef.current?.readyState === WebSocket.OPEN) {
      isTypingRef.current = true;
      wsRef.current.send(JSON.stringify({ type: 'typing' }));
    }

    if (typingTimerRef.current) clearTimeout(typingTimerRef.current);
    typingTimerRef.current = setTimeout(() => {
      isTypingRef.current = false;
    }, TYPING_TIMEOUT);
  };

  // ── File upload ─────────────────────────────────────────────────────────

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const attachment = await chatApi.uploadFile(woId, file);
      await chatApi.sendMessage(woId, {
        text: attachment.file_name,
        message_type: 'image',
        attachments: [attachment],
      });
    } catch (err) {
      console.error('Failed to upload file', err);
    }

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  // ── Reactions ───────────────────────────────────────────────────────────

  const sendReaction = (emoji: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          type: 'reaction',
          payload: { message_id: messages[messages.length - 1]?.id, emoji },
        }),
      );
    }
    setShowEmojiPicker(false);
  };

  // ── Read receipts ───────────────────────────────────────────────────────

  const markAsRead = useCallback(
    (messageId: string) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: 'read_receipt',
            payload: { message_id: messageId },
          }),
        );
      }
    },
    [],
  );

  useEffect(() => {
    if (messages.length > 0 && user) {
      const lastMsg = messages[messages.length - 1];
      if (lastMsg.user_id !== user.id) {
        markAsRead(lastMsg.id);
      }
    }
  }, [messages, user, markAsRead]);

  // ── Mention selection ───────────────────────────────────────────────────

  const insertMention = (mentionUser: MentionUser) => {
    const cursorPos = inputRef.current?.selectionStart || 0;
    const textBeforeCursor = input.slice(0, cursorPos);
    const textAfterCursor = input.slice(cursorPos);
    const atMatch = textBeforeCursor.match(/@(\w*)$/);

    if (atMatch) {
      const newText =
        textBeforeCursor.slice(0, textBeforeCursor.length - atMatch[0].length) +
        `@${mentionUser.name} ` +
        textAfterCursor;
      setInput(newText);
    }
    setShowMentions(false);
    inputRef.current?.focus();
  };

  const handleMentionKeyDown = (e: React.KeyboardEvent) => {
    if (!showMentions) return;

    const filtered = activeUsers.filter((u) =>
      u.name.toLowerCase().includes(mentionSearch.toLowerCase()),
    );

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setMentionCursor((prev) => Math.min(prev + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setMentionCursor((prev) => Math.max(prev - 1, 0));
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault();
      if (filtered[mentionCursor]) {
        insertMention(filtered[mentionCursor]);
      }
    } else if (e.key === 'Escape') {
      setShowMentions(false);
    }
  };

  // ── Scroll ──────────────────────────────────────────────────────────────

  const scrollToBottom = () => {
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 50);
  };

  // ── Load more (scroll up) ───────────────────────────────────────────────

  const handleScrollUp = useCallback(
    async (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      if (target.scrollTop === 0 && hasMore && !loading) {
        await loadHistory();
      }
    },
    [hasMore, loading, loadHistory],
  );

  // ── Keyboard ────────────────────────────────────────────────────────────

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
    handleMentionKeyDown(e);
  };

  // ── Render helpers ──────────────────────────────────────────────────────

  const formatTime = (iso: string) => {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    const today = new Date();
    if (d.toDateString() === today.toDateString()) return formatTime(iso);
    return (
      d.toLocaleDateString([], { day: 'numeric', month: 'short' }) + ' ' + formatTime(iso)
    );
  };

  const renderMessage = (msg: ChatMessage, idx: number) => {
    const isOwn = msg.user_id === user?.id;
    const prevMsg = idx > 0 ? messages[idx - 1] : null;
    const showAvatar = !prevMsg || prevMsg.user_id !== msg.user_id;

    return (
      <div
        key={msg.id}
        className={`flex ${isOwn ? 'justify-end' : 'justify-start'} mb-3`}
      >
        <div
          className={`max-w-[75%] rounded-lg px-3 py-2 ${
            isOwn
              ? 'bg-blue-500 text-white rounded-br-none'
              : 'bg-gray-100 text-gray-900 rounded-bl-none'
          }`}
        >
          {!isOwn && showAvatar && (
            <p className="text-xs font-semibold text-blue-600 mb-1">{msg.user_name}</p>
          )}

          {msg.message_type === 'system' && (
            <p className="text-xs italic text-gray-500 text-center">{msg.text}</p>
          )}

          {msg.message_type === 'text' && (
            <p className="text-sm whitespace-pre-wrap break-words">
              {renderTextWithMentions(msg.text)}
            </p>
          )}

          {msg.message_type === 'image' && msg.attachments && msg.attachments.length > 0 && (
            <div className="mt-1">
              <p className="text-xs opacity-75">{msg.attachments[0].file_name}</p>
              {msg.attachments[0].mime_type.startsWith('image/') && (
                <img
                  src={msg.attachments[0].storage_path}
                  alt={msg.attachments[0].file_name}
                  className="max-w-full rounded mt-1 max-h-48 object-cover"
                  loading="lazy"
                />
              )}
            </div>
          )}

          <div
            className={`flex items-center gap-1 mt-1 ${isOwn ? 'justify-end' : 'justify-start'}`}
          >
            <span className="text-[10px] opacity-60">{formatDate(msg.created_at)}</span>
            {isOwn && msg.read_by && msg.read_by.length > 0 && (
              <span className="text-[10px] text-blue-200">
                <Check size={12} className="inline" />
              </span>
            )}
          </div>
        </div>
      </div>
    );
  };

  const renderTextWithMentions = (text: string) => {
    const parts = text.split(/(@\w+)/g);
    return parts.map((part, i) => {
      if (part.startsWith('@')) {
        return (
          <span key={i} className="text-blue-400 font-semibold">
            {part}
          </span>
        );
      }
      return <span key={i}>{part}</span>;
    });
  };

  const filteredMentions = activeUsers.filter((u) =>
    u.name.toLowerCase().includes(mentionSearch.toLowerCase()),
  );

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/40">
      <div className="bg-white rounded-t-xl sm:rounded-xl w-full sm:max-w-lg sm:mx-4 h-[70vh] sm:h-[60vh] flex flex-col shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <div className="flex items-center gap-2">
            <MessageCircle size={18} className="text-blue-500" />
            <h3 className="font-semibold text-sm">
              {woId.length > 20 ? woId.slice(0, 20) + '...' : woId}
            </h3>
            <span className="text-xs text-gray-400">
              {activeUsers.length > 0 ? `${activeUsers.length} online` : ''}
            </span>
          </div>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 rounded-full transition-colors"
            aria-label="Close chat"
          >
            <X size={18} />
          </button>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-4 py-3" onScroll={handleScrollUp}>
          {hasMore && !loading && (
            <div className="text-center py-2">
              <button
                onClick={loadHistory}
                className="text-xs text-blue-500 hover:text-blue-600"
              >
                <ChevronDown size={14} className="inline mr-1" />
                Load older messages
              </button>
            </div>
          )}

          {loading && (
            <div className="flex justify-center py-4">
              <Loader2 size={20} className="animate-spin text-gray-400" />
            </div>
          )}

          {!loading && messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full text-gray-400">
              <MessageCircle size={40} className="mb-2 opacity-50" />
              <p className="text-sm">No messages yet</p>
              <p className="text-xs">Send the first message</p>
            </div>
          )}

          {messages.map((msg, idx) => renderMessage(msg, idx))}

          {typingUsers.length > 0 && (
            <div className="flex items-center gap-2 text-xs text-gray-400 italic py-1">
              <div className="flex gap-0.5">
                <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" />
                <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" />
                <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" />
              </div>
              {typingUsers.join(', ')} typing...
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* @mentions popup */}
        {showMentions && filteredMentions.length > 0 && (
          <div className="absolute bottom-20 left-4 right-4 bg-white border border-gray-200 rounded-lg shadow-lg max-h-32 overflow-y-auto z-10">
            {filteredMentions.map((u, i) => (
              <button
                key={u.id}
                className={`w-full text-left px-3 py-2 text-sm hover:bg-blue-50 ${
                  i === mentionCursor ? 'bg-blue-50' : ''
                }`}
                onClick={() => insertMention(u)}
              >
                @{u.name}
              </button>
            ))}
          </div>
        )}

        {/* Emoji picker */}
        {showEmojiPicker && (
          <div className="absolute bottom-20 right-4 bg-white border border-gray-200 rounded-lg shadow-lg p-2 z-10">
            <div className="flex gap-1">
              {EMOJI_LIST.map((emoji) => (
                <button
                  key={emoji}
                  className="p-1 hover:bg-gray-100 rounded text-lg"
                  onClick={() => sendReaction(emoji)}
                >
                  {emoji}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Input area */}
        <div className="border-t border-gray-200 p-3">
          <div className="flex items-end gap-2">
            <button
              onClick={() => fileInputRef.current?.click()}
              className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-400 hover:text-gray-600"
              aria-label="Attach file"
            >
              <Upload size={18} />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,.pdf,.doc,.docx,.xls,.xlsx"
              className="hidden"
              onChange={handleFileUpload}
            />

            <div className="flex-1 relative">
              <textarea
                ref={inputRef}
                value={input}
                onChange={handleInputChange}
                onKeyDown={handleKeyDown}
                placeholder="Type a message... (@ to mention)"
                rows={1}
                maxLength={MAX_MESSAGE_LENGTH}
                className="w-full resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                style={{ minHeight: '36px', maxHeight: '120px' }}
              />
            </div>

            <button
              onClick={() => setShowEmojiPicker(!showEmojiPicker)}
              className="p-2 hover:bg-gray-100 rounded-full transition-colors text-gray-400 hover:text-gray-600"
              aria-label="Add reaction"
            >
              <Heart size={18} />
            </button>

            <button
              onClick={sendMessage}
              disabled={!input.trim() || sending}
              className="p-2 bg-blue-500 text-white rounded-full hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              aria-label="Send message"
            >
              {sending ? <Loader2 size={18} className="animate-spin" /> : <Send size={18} />}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
