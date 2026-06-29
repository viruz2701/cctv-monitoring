// ═══════════════════════════════════════════════════════════════════════
// VideoTutorialCard — UX-14.3.5: Onboarding Video Tutorials
//
// Карточка с плейсхолдером видео:
//   - Серый прямоугольник с иконкой Play
//   - Название, описание, длительность
//   - При клике открывает Modal с iframe/placeholder
//   - "Coming soon" для видео без ссылки
// ═══════════════════════════════════════════════════════════════════════

import React, { useState } from 'react';
import { Play, Clock, Film, X } from './Icons';
import { Modal } from './Modal';

export interface TutorialVideo {
  id: string;
  title: string;
  description: string;
  duration: string; // e.g. "3:45"
  category: string;
  /** YouTube embed URL или null, если видео ещё не готово */
  videoUrl: string | null;
  /** Дата публикации */
  publishedAt?: string;
}

interface VideoTutorialCardProps {
  video: TutorialVideo;
}

export function VideoTutorialCard({ video }: VideoTutorialCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const isComingSoon = !video.videoUrl;

  return (
    <>
      <button
        onClick={() => !isComingSoon && setIsOpen(true)}
        className={`group w-full text-left rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden transition-all ${
          isComingSoon
            ? 'opacity-70 cursor-default'
            : 'hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-700 hover:-translate-y-0.5 cursor-pointer'
        }`}
        aria-label={`${isComingSoon ? 'Coming soon: ' : ''}${video.title}`}
      >
        {/* Video Placeholder */}
        <div className="relative aspect-video bg-slate-100 dark:bg-slate-800 flex items-center justify-center overflow-hidden">
          {isComingSoon ? (
            <div className="flex flex-col items-center gap-2 text-slate-400 dark:text-slate-500">
              <Film size={40} aria-hidden="true" />
              <span className="text-xs font-medium uppercase tracking-wider">Coming Soon</span>
            </div>
          ) : (
            <>
              <div className="absolute inset-0 bg-gradient-to-br from-slate-200 to-slate-300 dark:from-slate-700 dark:to-slate-800" />
              <div className="relative flex items-center justify-center w-16 h-16 bg-black/40 dark:bg-black/50 rounded-full group-hover:bg-black/60 group-hover:scale-110 transition-all">
                <Play size={28} className="text-white ml-1" aria-hidden="true" />
              </div>
            </>
          )}
        </div>

        {/* Info */}
        <div className="p-4">
          <div className="flex items-start justify-between gap-2">
            <h3 className={`text-sm font-semibold ${isComingSoon ? 'text-slate-400 dark:text-slate-500' : 'text-slate-900 dark:text-white'}`}>
              {video.title}
            </h3>
          </div>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400 line-clamp-2">
            {video.description}
          </p>
          <div className="mt-3 flex items-center gap-3 text-xs text-slate-400 dark:text-slate-500">
            <span className="flex items-center gap-1">
              <Clock size={12} aria-hidden="true" />
              {video.duration}
            </span>
            <span className="px-1.5 py-0.5 bg-slate-100 dark:bg-slate-700 rounded text-[10px] font-medium uppercase">
              {video.category}
            </span>
          </div>
        </div>
      </button>

      {/* Video Modal */}
      {isOpen && video.videoUrl && (
        <Modal isOpen={isOpen} onClose={() => setIsOpen(false)} title={video.title} size="lg">
          <div className="aspect-video bg-slate-900 rounded-lg flex items-center justify-center overflow-hidden">
            {video.videoUrl.includes('youtube.com') || video.videoUrl.includes('youtu.be') ? (
              <iframe
                src={video.videoUrl}
                title={video.title}
                className="w-full h-full"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
              />
            ) : (
              <div className="flex flex-col items-center gap-3 text-slate-400">
                <Film size={48} aria-hidden="true" />
                <p className="text-sm">Video not available for preview</p>
              </div>
            )}
          </div>
          <div className="mt-4">
            <p className="text-sm text-slate-600 dark:text-slate-400">
              {video.description}
            </p>
            <p className="mt-2 text-xs text-slate-400 dark:text-slate-500">
              Duration: {video.duration}
            </p>
          </div>
        </Modal>
      )}
    </>
  );
}
