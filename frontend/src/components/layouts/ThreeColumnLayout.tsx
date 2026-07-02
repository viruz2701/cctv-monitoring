// ═══════════════════════════════════════════════════════════════════════
// ThreeColumnLayout — трёхколоночный макет для Device Detail
// UX-2.1: Three-Column Layout Pattern
//
// Feature Flag: three_column_detail_layout
//
// Desktop:
//   Left (240px) — breadcrumbs, status, quick actions
//   Center (flex) — tabs (Overview, Live View, History, Documents)
//   Right (320px) — metadata, SLA timer, assignee, audit log, actions
//
// Mobile (<1024px): columns stack vertically
// ═══════════════════════════════════════════════════════════════════════

import React, { useState } from 'react';
import { ChevronDown, ChevronUp } from '../ui/Icons';

// ── Types ────────────────────────────────────────────────────────────

export interface ThreeColumnLayoutProps {
  /** Sticky header (device name, status badges, actions) */
  header: React.ReactNode;

  /** Left panel (240px, desktop) — breadcrumbs, status, quick actions */
  left: React.ReactNode;

  /** Center panel (flex) — tab content */
  center: React.ReactNode;

  /** Right panel (320px, desktop) — metadata, SLA, audit log */
  right: React.ReactNode;

  /** Labels for mobile accordion sections */
  leftHeader?: string;
  centerHeader?: string;
  rightHeader?: string;
}

// ── Component ────────────────────────────────────────────────────────

export const ThreeColumnLayout: React.FC<ThreeColumnLayoutProps> = ({
  header,
  left,
  center,
  right,
  leftHeader = 'Панель',
  centerHeader = 'Основное',
  rightHeader = 'Детали',
}) => {
  const [mobileSection, setMobileSection] = useState<string | null>('center');

  const toggleSection = (section: string) => {
    setMobileSection((prev) => (prev === section ? null : section));
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      {/* ═══ Sticky Header ═══ */}
      <div className="sticky top-0 z-30 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 shadow-sm">
        {header}
      </div>

      {/* ═══ Desktop: 3 columns ═══ */}
      <div className="hidden lg:grid lg:grid-cols-[240px_1fr_320px] gap-6 p-6 max-w-7xl mx-auto">
        {/* Left Column — 240px */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)] sticky top-24">
          {left}
        </div>

        {/* Center Column — flex */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)] min-w-0">
          {center}
        </div>

        {/* Right Column — 320px */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)] sticky top-24">
          {right}
        </div>
      </div>

      {/* ═══ Mobile: Accordion ═══ */}
      <div className="lg:hidden p-4 space-y-3">
        <MobileSection
          title={leftHeader}
          isOpen={mobileSection === 'left'}
          onToggle={() => toggleSection('left')}
        >
          {left}
        </MobileSection>

        <MobileSection
          title={centerHeader}
          isOpen={mobileSection === 'center'}
          onToggle={() => toggleSection('center')}
        >
          {center}
        </MobileSection>

        <MobileSection
          title={rightHeader}
          isOpen={mobileSection === 'right'}
          onToggle={() => toggleSection('right')}
        >
          {right}
        </MobileSection>
      </div>
    </div>
  );
};

// ── Mobile accordion sub-component ───────────────────────────────────

interface MobileSectionProps {
  title: string;
  isOpen: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

const MobileSection: React.FC<MobileSectionProps> = ({
  title,
  isOpen,
  onToggle,
  children,
}) => (
  <div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
    <button
      onClick={onToggle}
      className="w-full flex items-center justify-between px-4 py-3 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
    >
      <span>{title}</span>
      {isOpen ? (
        <ChevronUp className="w-4 h-4 text-slate-400" />
      ) : (
        <ChevronDown className="w-4 h-4 text-slate-400" />
      )}
    </button>
    <div
      className="transition-all duration-300 ease-in-out overflow-hidden"
      style={{
        maxHeight: isOpen ? '9999px' : '0px',
        opacity: isOpen ? 1 : 0,
      }}
    >
      <div className="p-4 border-t border-slate-100 dark:border-slate-800">
        {children}
      </div>
    </div>
  </div>
);
