// ═══════════════════════════════════════════════════════════════════════
// PresenceIndicators — P3-NICE.1: Real-time Collaboration
//
// Показывает:
//   - Аватары пользователей, просматривающих/редактирующих WO
//   - Индикатор "editing" (жёлтый) vs "viewing" (зелёный)
//   - Conflict warning при одновременном редактировании
//   - Cursor positions (опционально)
//
// Соответствие:
//   - WCAG 2.1 SC 1.4.1 (Use of Color — иконки + текст)
//   - WCAG 2.1 SC 2.5.3 (Label in Name для аватаров)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Users, Edit3, Eye, AlertTriangle, Wifi, WifiOff } from 'lucide-react';
import type { PresenceInfo, CursorInfo } from '../../services/collaboration';

// ═══ Avatar color palette ═══════════════════════════════════════════════

const AVATAR_COLORS = [
  'bg-blue-500',
  'bg-emerald-500',
  'bg-violet-500',
  'bg-amber-500',
  'bg-rose-500',
  'bg-cyan-500',
  'bg-pink-500',
  'bg-teal-500',
];

function getAvatarColor(userId: string): string {
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = ((hash << 5) - hash) + userId.charCodeAt(i);
    hash |= 0;
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);
}

// ═══ Types ══════════════════════════════════════════════════════════════

interface PresenceIndicatorsProps {
  presences: PresenceInfo[];
  cursors?: CursorInfo[];
  hasConflict?: boolean;
  conflictUsers?: string[];
  isConnected?: boolean;
  currentUserId?: string;
}

interface CursorOverlayProps {
  cursors: CursorInfo[];
  currentUserId?: string;
  containerRef?: React.RefObject<HTMLDivElement | null>;
}

// ═══ PresenceAvatars — миниатюры пользователей ═════════════════════════

export function PresenceAvatars({
  presences,
  hasConflict,
  conflictUsers,
  isConnected,
  currentUserId,
}: PresenceIndicatorsProps) {
  const { t } = useTranslation();

  const otherUsers = presences.filter((p) => p.user_id !== currentUserId);
  const displayUsers = otherUsers.slice(0, 5);
  const overflow = otherUsers.length - 5;

  return (
    <div className="flex items-center gap-2" role="region" aria-label={t('collab.presence') || 'Присутствие'}>
      {/* Connection indicator */}
      <div className="flex items-center gap-1 mr-1">
        {isConnected ? (
          <Wifi className="w-3 h-3 text-green-500" aria-label={t('collab.connected') || 'Подключено'} />
        ) : (
          <WifiOff className="w-3 h-3 text-slate-400" aria-label={t('collab.disconnected') || 'Отключено'} />
        )}
      </div>

      {/* Conflict warning */}
      {hasConflict && conflictUsers && conflictUsers.length > 0 && (
        <div
          className="flex items-center gap-1 px-2 py-1 bg-amber-50 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-700 rounded-md"
          role="alert"
        >
          <AlertTriangle className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
          <span className="text-xs text-amber-700 dark:text-amber-300 font-medium">
            {conflictUsers.join(', ')} {conflictUsers.length === 1 ? t('collab.editing') || 'редактирует' : t('collab.editing_plural') || 'редактируют'}
          </span>
        </div>
      )}

      {/* Presence avatars */}
      {displayUsers.length > 0 && (
        <div className="flex -space-x-1.5" role="list" aria-label={t('collab.active_users') || 'Активные пользователи'}>
          {displayUsers.map((p) => (
            <div
              key={p.user_id}
              className="group relative"
              role="listitem"
            >
              <div
                className={`
                  w-7 h-7 rounded-full flex items-center justify-center text-white text-xs font-medium
                  ring-2 ring-white dark:ring-slate-800 transition-shadow
                  ${getAvatarColor(p.user_id)}
                  ${p.action === 'editing' ? 'ring-amber-400 dark:ring-amber-500' : ''}
                `}
                title={`${p.user_name} — ${p.action === 'editing' ? t('collab.editing') || 'редактирует' : t('collab.viewing') || 'просматривает'}`}
                aria-label={`${p.user_name} — ${p.action === 'editing' ? t('collab.editing') || 'редактирует' : t('collab.viewing') || 'просматривает'}`}
              >
                {getInitials(p.user_name)}
              </div>
              {/* Action indicator dot */}
              <span
                className={`
                  absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white dark:border-slate-800
                  ${p.action === 'editing' ? 'bg-amber-400' : 'bg-green-400'}
                `}
                aria-hidden="true"
              />
              {/* Tooltip */}
              <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block z-50">
                <div className="bg-slate-900 dark:bg-slate-700 text-white text-xs px-2 py-1 rounded whitespace-nowrap shadow-lg">
                  <div className="font-medium">{p.user_name}</div>
                  <div className="text-slate-300 text-[10px]">
                    {p.action === 'editing' ? (
                      <><Edit3 className="w-3 h-3 inline mr-0.5" /> {t('collab.editing') || 'Редактирует'}</>
                    ) : (
                      <><Eye className="w-3 h-3 inline mr-0.5" /> {t('collab.viewing') || 'Просматривает'}</>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}

          {/* Overflow badge */}
          {overflow > 0 && (
            <div className="w-7 h-7 rounded-full flex items-center justify-center bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 text-xs font-medium ring-2 ring-white dark:ring-slate-800">
              +{overflow}
            </div>
          )}
        </div>
      )}

      {/* Empty state */}
      {displayUsers.length === 0 && !hasConflict && (
        <span className="text-xs text-slate-400 dark:text-slate-500 flex items-center gap-1">
          <Users className="w-3.5 h-3.5" />
          {t('collab.no_users') || 'Нет других пользователей'}
        </span>
      )}
    </div>
  );
}

// ═══ CursorOverlay — отображение курсоров других пользователей ═════════

export function CursorOverlay({
  cursors,
  currentUserId,
  containerRef,
}: CursorOverlayProps) {
  const otherCursors = cursors.filter((c) => c.user_id !== currentUserId);

  if (otherCursors.length === 0) return null;

  const container = containerRef?.current;
  const rect = container?.getBoundingClientRect();

  return (
    <div className="pointer-events-none absolute inset-0 z-50" aria-hidden="true">
      {otherCursors.map((c) => (
        <div
          key={`${c.user_id}-${c.field}`}
          className="absolute flex items-start gap-1 transition-all duration-150 ease-out"
          style={{
            top: `${c.line * 24}px`, // approximate line height
            left: `${c.col * 8}px`,   // approximate char width
          }}
        >
          {/* Cursor caret */}
          <svg width="12" height="16" viewBox="0 0 12 16" className="text-amber-500">
            <path
              d="M2 0 L10 6 L6 8 L8 14 L4 16 L2 10 L0 8 Z"
              fill="currentColor"
              opacity="0.8"
            />
          </svg>
          {/* Label */}
          <span
            className="text-[10px] font-medium text-white px-1 py-0.5 rounded-sm whitespace-nowrap"
            style={{ backgroundColor: getAvatarColor(c.user_id) }}
          >
            {c.user_name}
          </span>
        </div>
      ))}
    </div>
  );
}

// ═══ CollaborationStatusBar — статус-бар для шапки WO ═══════════════════

interface CollaborationStatusBarProps {
  presences: PresenceInfo[];
  hasConflict: boolean;
  conflictUsers: string[];
  isConnected: boolean;
  currentUserId: string;
  onEditingToggle?: (editing: boolean) => void;
  isEditing?: boolean;
}

export function CollaborationStatusBar({
  presences,
  hasConflict,
  conflictUsers,
  isConnected,
  currentUserId,
  onEditingToggle,
  isEditing,
}: CollaborationStatusBarProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between px-3 py-1.5 bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700 text-xs">
      <PresenceAvatars
        presences={presences}
        hasConflict={hasConflict}
        conflictUsers={conflictUsers}
        isConnected={isConnected}
        currentUserId={currentUserId}
      />

      {/* Editing toggle */}
      {onEditingToggle && (
        <button
          onClick={() => onEditingToggle(!isEditing)}
          className={`
            flex items-center gap-1.5 px-2 py-1 rounded-md transition-colors text-xs font-medium
            ${isEditing
              ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300 hover:bg-amber-200 dark:hover:bg-amber-900/50'
              : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
            }
            ${hasConflict ? 'animate-pulse' : ''}
          `}
          aria-pressed={isEditing}
          aria-label={isEditing ? t('collab.stop_editing') || 'Завершить редактирование' : t('collab.start_editing') || 'Начать редактирование'}
        >
          {isEditing ? (
            <>
              <Edit3 className="w-3 h-3" />
              <span>{t('collab.editing') || 'Редактирую'}</span>
            </>
          ) : (
            <>
              <Eye className="w-3 h-3" />
              <span>{t('collab.viewing') || 'Просмотр'}</span>
            </>
          )}
        </button>
      )}
    </div>
  );
}
