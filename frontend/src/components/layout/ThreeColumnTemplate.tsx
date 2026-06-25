import React, { useState } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';

interface ThreeColumnTemplateProps {
  /** Sticky header (e.g. WODetailHeader) */
  header: React.ReactNode;
  /** Left panel (25% on desktop) */
  left: React.ReactNode;
  /** Center panel (50% on desktop) */
  center: React.ReactNode;
  /** Right panel (25% on desktop) */
  right: React.ReactNode;
  /** Collapsible label for left panel on mobile */
  leftHeader?: string;
  /** Collapsible label for center panel on mobile */
  centerHeader?: string;
  /** Collapsible label for right panel on mobile */
  rightHeader?: string;
}

/**
 * Three-column layout template (Atlas CMMS pattern).
 *
 * Desktop: 3 independent scrolling columns (25% / 50% / 25%).
 * Mobile (<1024px): single column with collapsible accordion sections.
 */
export const ThreeColumnTemplate: React.FC<ThreeColumnTemplateProps> = ({
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
    setMobileSection(prev => (prev === section ? null : section));
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      {/* ═══ Sticky Header ═══ */}
      <div className="sticky top-0 z-30 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 shadow-sm">
        {header}
      </div>

      {/* ═══ Desktop: 3 columns ═══ */}
      <div className="hidden lg:grid lg:grid-cols-[1fr_2fr_1fr] gap-6 p-6 max-w-7xl mx-auto">
        {/* Left Column */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)] sticky top-24">
          {left}
        </div>

        {/* Center Column */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)]">
          {center}
        </div>

        {/* Right Column */}
        <div className="overflow-y-auto space-y-4 max-h-[calc(100vh-8rem)] sticky top-24">
          {right}
        </div>
      </div>

      {/* ═══ Mobile: Accordion ═══ */}
      <div className="lg:hidden p-4 space-y-3">
        {/* Left section */}
        <MobileSection
          title={leftHeader}
          isOpen={mobileSection === 'left'}
          onToggle={() => toggleSection('left')}
        >
          {left}
        </MobileSection>

        {/* Center section (open by default) */}
        <MobileSection
          title={centerHeader}
          isOpen={mobileSection === 'center'}
          onToggle={() => toggleSection('center')}
        >
          {center}
        </MobileSection>

        {/* Right section */}
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
