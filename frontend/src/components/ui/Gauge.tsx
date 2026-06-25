import React from 'react';

export interface GaugeProps {
  value: number;
  max?: number;
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  color?: string;
  thresholds?: { value: number; color: string; label?: string }[];
  showValue?: boolean;
  unit?: string;
  className?: string;
}

// P0-4.1: SLA-стандартные пороги — green ≥95%, yellow 80–94%, orange 60–79%, red <60%
const DEFAULT_THRESHOLDS = [
  { value: 95, color: '#16a34a', label: '≥95%' },
  { value: 80, color: '#eab308', label: '80–94%' },
  { value: 60, color: '#f97316', label: '60–79%' },
  { value: 0, color: '#dc2626', label: '<60%' },
];

export function Gauge({
  value,
  max = 100,
  label,
  size = 'md',
  thresholds,
  showValue = true,
  unit = '%',
  className = '',
}: GaugeProps) {
  // P0-4.1: Mount animation via stroke-dasharray
  const [animatedPct, setAnimatedPct] = React.useState(0);
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
    const timer = setTimeout(() => setAnimatedPct(Math.min(100, Math.max(0, (value / max) * 100))), 50);
    return () => clearTimeout(timer);
  }, [value, max]);

  const pct = animatedPct;
  const activeThresholds = thresholds ?? DEFAULT_THRESHOLDS;
  const radius = size === 'lg' ? 52 : size === 'sm' ? 34 : 42;
  const stroke = size === 'lg' ? 10 : size === 'sm' ? 6 : 8;
  const circumference = 2 * Math.PI * radius;
  const offset = mounted ? circumference - (pct / 100) * circumference : circumference;

  const sizeClasses = {
    sm: 'w-24 h-24',
    md: 'w-32 h-32',
    lg: 'w-40 h-40',
  };

  const fontSize = {
    sm: 'text-lg',
    md: 'text-2xl',
    lg: 'text-3xl',
  };

  const svgSize = radius * 2 + stroke * 2;
  const center = svgSize / 2;

  const getColor = (): string => {
    for (let i = activeThresholds.length - 1; i >= 0; i--) {
      if (pct >= activeThresholds[i].value) {
        return activeThresholds[i].color;
      }
    }
    return activeThresholds[activeThresholds.length - 1]?.color ?? '#16a34a';
  };

  const fillColor = getColor();

  return (
    <div className={`flex flex-col items-center ${className}`}>
      {label && (
        <p className="text-sm font-medium text-slate-600 dark:text-slate-300 mb-2">{label}</p>
      )}
      <div className={`relative ${sizeClasses[size]}`}>
        <svg
          viewBox={`0 0 ${svgSize} ${svgSize}`}
          className="w-full h-full -rotate-90"
        >
          {/* Track circle */}
          <circle
            cx={center}
            cy={center}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={stroke}
            className="text-slate-200 dark:text-slate-700"
          />
          {/* Arc with mount animation */}
          <circle
            cx={center}
            cy={center}
            r={radius}
            fill="none"
            stroke={fillColor}
            strokeWidth={stroke}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            className="transition-all duration-1000 ease-out"
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {showValue && (
            <span className={`font-bold ${fontSize[size]} text-slate-900 dark:text-white`}>
              {Math.round(pct)}
              <span className="text-xs text-slate-500 dark:text-slate-400">{unit}</span>
            </span>
          )}
        </div>
      </div>

      {activeThresholds.length > 0 && (
        <div className="flex gap-3 mt-2">
          {activeThresholds.map((t, i) => (
            <div key={i} className="flex items-center gap-1">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: t.color }} />
              <span className="text-xs text-slate-500 dark:text-slate-400">{t.label || `${t.value}%`}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}