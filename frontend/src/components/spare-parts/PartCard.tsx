// ═══════════════════════════════════════════════════════════════════════
// PartCard — карточка запчасти с изображением, stock индикацией и QR
// Shelf.nu inspired: stock > min_stock*2 → 🟢, > min_stock → 🟡, ≤ → 🔴, =0 → 🔴+icon
// ═══════════════════════════════════════════════════════════════════════

import React, { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { Modal } from '../ui/Modal';
import { Badge } from '../ui/Badge';
import {
  Package,
  AlertTriangle,
  ImageOff,
  QrCode,
  Wrench,
  MapPin,
  DollarSign,
  Truck,
} from 'lucide-react';

// ── Types ──────────────────────────────────────────────────────────────

export interface PartCardPart {
  id: string;
  name: string;
  sku: string;
  category?: string;
  stock: number;
  min_stock: number;
  location?: string;
  cost?: number;
  supplier?: string;
  image_url?: string;
}

interface PartCardProps {
  part: PartCardPart;
  onView?: (id: string) => void;
  onOrder?: (id: string) => void;
  /** Показать low-stock акцент (рамка + иконка ⚠️) */
  lowStockAccent?: boolean;
}

// ── Helpers ────────────────────────────────────────────────────────────

type StockLevel = 'ok' | 'low' | 'critical' | 'out';

function getStockLevel(stock: number, minStock: number): StockLevel {
  // Shelf.nu pattern: stock > min_stock*2 → green, stock > min_stock → yellow,
  // stock > 0 && stock <= min_stock → red, stock = 0 → red with icon
  if (stock <= 0) return 'out';
  if (stock > minStock * 2) return 'ok';
  if (stock > minStock) return 'low';
  return 'critical';
}

const stockConfig: Record<StockLevel, { bg: string; text: string; icon: React.ReactNode; label: string }> = {
  ok: {
    bg: 'bg-emerald-100 dark:bg-emerald-900/30',
    text: 'text-emerald-700 dark:text-emerald-400',
    icon: <Package size={16} className="text-emerald-600 dark:text-emerald-400" />,
    label: 'В наличии',
  },
  low: {
    bg: 'bg-amber-100 dark:bg-amber-900/30',
    text: 'text-amber-700 dark:text-amber-400',
    icon: <Package size={16} className="text-amber-600 dark:text-amber-400" />,
    label: 'Мало',
  },
  critical: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-700 dark:text-red-400',
    icon: <AlertTriangle size={16} className="text-red-600 dark:text-red-400" />,
    label: 'Критично',
  },
  out: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-700 dark:text-red-400',
    icon: <AlertTriangle size={16} className="text-red-600 dark:text-red-400" />,
    label: 'Нет в наличии',
  },
};

// ── Component ──────────────────────────────────────────────────────────

export function PartCard({ part, onView, onOrder, lowStockAccent = false }: PartCardProps) {
  const [showQR, setShowQR] = useState(false);
  const [imgError, setImgError] = useState(false);

  const level = getStockLevel(part.stock, part.min_stock);
  const cfg = stockConfig[level];
  const isLow = level === 'low' || level === 'critical' || level === 'out';
  const isOut = level === 'out';

  const qrValue = JSON.stringify({
    id: part.id,
    sku: part.sku,
    name: part.name,
  });

  return (
    <>
      <div
        className={`relative rounded-xl border bg-white dark:bg-slate-800 shadow-sm transition-all duration-200 hover:shadow-md group ${
          lowStockAccent && isLow
            ? 'border-red-300 dark:border-red-700 ring-1 ring-red-200 dark:ring-red-800'
            : 'border-slate-200 dark:border-slate-700'
        }`}
      >
        {/* Low-stock warning badge */}
        {lowStockAccent && isLow && (
          <div className="absolute -top-2 -right-2 z-10">
            <span className="flex items-center gap-1 px-2 py-0.5 bg-red-500 text-white text-xs font-bold rounded-full shadow-sm">
              <AlertTriangle className="w-3 h-3" />
              {level === 'out' ? 'Out' : 'Low'}
            </span>
          </div>
        )}

        {/* Image / Placeholder */}
        <div
          className="relative h-36 rounded-t-xl bg-slate-100 dark:bg-slate-700 overflow-hidden cursor-pointer"
          onClick={() => onView?.(part.id)}
        >
          {part.image_url && !imgError ? (
            <img
              src={part.image_url}
              alt={part.name}
              className="w-full h-full object-cover"
              onError={() => setImgError(true)}
              loading="lazy"
            />
          ) : (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <ImageOff size={32} className="mx-auto text-slate-300 dark:text-slate-500" />
                <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">No image</p>
              </div>
            </div>
          )}

          {/* QR button overlay */}
          <button
            onClick={(e) => { e.stopPropagation(); setShowQR(true); }}
            className="absolute top-2 right-2 p-1.5 bg-white/90 dark:bg-slate-900/90 rounded-lg shadow-sm hover:bg-white dark:hover:bg-slate-900 transition-colors opacity-0 group-hover:opacity-100"
            title="Show QR Code"
            aria-label="Show QR Code"
          >
            <QrCode size={16} className="text-slate-600 dark:text-slate-300" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4">
          {/* Header: name + SKU */}
          <div className="mb-3 cursor-pointer" onClick={() => onView?.(part.id)}>
            <h3 className="font-semibold text-slate-900 dark:text-white text-sm leading-tight line-clamp-2">
              {part.name}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">
              SKU: {part.sku}
            </p>
          </div>

          {/* Stock level indicator */}
          <div className="flex items-center justify-between mb-3">
            <div className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${cfg.bg} ${cfg.text}`}>
              {cfg.icon}
              {cfg.label}
            </div>
            <div className="text-right">
              <span className={`text-lg font-bold ${level === 'out' || level === 'critical' ? 'text-red-600 dark:text-red-400' : level === 'low' ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400'}`}>
                {part.stock}
              </span>
              <span className="text-xs text-slate-400 ml-1">/ {part.min_stock} min</span>
            </div>
          </div>

          {/* Stock bar */}
          <div className="w-full h-1.5 bg-slate-100 dark:bg-slate-700 rounded-full overflow-hidden mb-3">
            <div
              className={`h-full rounded-full transition-all duration-300 ${
                level === 'out' || level === 'critical'
                  ? 'bg-red-500'
                  : level === 'low'
                  ? 'bg-amber-500'
                  : 'bg-emerald-500'
              }`}
              style={{ width: `${Math.min(100, (part.stock / Math.max(part.min_stock * 2, 1)) * 100)}%` }}
            />
          </div>

          {/* Meta info */}
          <div className="space-y-1.5 mb-3">
            {part.category && (
              <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                <Wrench size={12} className="shrink-0" />
                <span className="truncate">{part.category}</span>
              </div>
            )}
            {part.location && (
              <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                <MapPin size={12} className="shrink-0" />
                <span className="truncate">{part.location}</span>
              </div>
            )}
            {part.supplier && (
              <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                <Truck size={12} className="shrink-0" />
                <span className="truncate">{part.supplier}</span>
              </div>
            )}
            {part.cost != null && (
              <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                <DollarSign size={12} className="shrink-0" />
                <span>${part.cost.toFixed(2)}</span>
              </div>
            )}
          </div>

          {/* Actions */}
          {onOrder && isLow && (
            <button
              onClick={(e) => { e.stopPropagation(); onOrder(part.id); }}
              className="w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
            >
              Заказать
            </button>
          )}
        </div>
      </div>

      {/* QR Code Modal */}
      <Modal isOpen={showQR} onClose={() => setShowQR(false)} title={`QR Code: ${part.name}`} size="sm">
        <div className="flex flex-col items-center py-4">
          <QRCodeSVG
            value={qrValue}
            size={200}
            level="M"
            includeMargin
            className="rounded-lg border border-slate-200 dark:border-slate-700"
          />
          <p className="mt-3 text-sm text-slate-500 dark:text-slate-400 font-mono">
            {part.sku}
          </p>
          <button
            onClick={() => {
              const svg = document.querySelector('.qr-code-modal svg');
              if (svg) {
                const svgData = new XMLSerializer().serializeToString(svg);
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                const img = new Image();
                img.onload = () => {
                  canvas.width = img.width;
                  canvas.height = img.height;
                  ctx?.drawImage(img, 0, 0);
                  const url = canvas.toDataURL('image/png');
                  const a = document.createElement('a');
                  a.href = url;
                  a.download = `qr-${part.sku}.png`;
                  a.click();
                };
                img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svgData)));
              }
            }}
            className="mt-3 px-4 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-800 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
          >
            Скачать QR
          </button>
        </div>
      </Modal>
    </>
  );
}
