import React from 'react';
import { Card, CardBody } from './Card';
import { Badge } from './Badge';
import { Package, AlertTriangle, CheckCircle, Wrench } from './Icons';

interface PartCardProps {
  part: {
    id: string;
    name: string;
    part_number?: string;
    category?: string;
    quantity: number;
    min_quantity?: number;
    unit?: string;
    price?: number;
    supplier?: string;
    last_ordered?: string;
    location?: string;
  };
  onView?: (id: string) => void;
  onOrder?: (id: string) => void;
}

export function PartCard({ part, onView, onOrder }: PartCardProps) {
  const isLowStock = part.min_quantity != null && part.quantity <= part.min_quantity;
  const isOutOfStock = part.quantity <= 0;

  const stockStatus = isOutOfStock ? 'out' : isLowStock ? 'low' : 'normal';

  return (
    <div
      className={`hover:shadow-md transition-shadow cursor-pointer`}
      onClick={() => onView?.(part.id)}
    >
    <Card
      className={isOutOfStock ? 'border-red-300 dark:border-red-800' : ''}
    >
      <CardBody>
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center gap-2">
            <div
              className={`p-2 rounded-lg ${
                stockStatus === 'out'
                  ? 'bg-red-100 dark:bg-red-900/30'
                  : stockStatus === 'low'
                  ? 'bg-amber-100 dark:bg-amber-900/30'
                  : 'bg-emerald-100 dark:bg-emerald-900/30'
              }`}
            >
              {stockStatus === 'out' ? (
                <AlertTriangle size={18} className="text-red-600 dark:text-red-400" />
              ) : stockStatus === 'low' ? (
                <Package size={18} className="text-amber-600 dark:text-amber-400" />
              ) : (
                <CheckCircle size={18} className="text-emerald-600 dark:text-emerald-400" />
              )}
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 dark:text-white text-sm">{part.name}</h3>
              {part.part_number && (
                <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">{part.part_number}</p>
              )}
            </div>
          </div>
          <Badge
            variant={stockStatus === 'out' ? 'danger' : stockStatus === 'low' ? 'warning' : 'success'}
            size="sm"
          >
            {stockStatus === 'out' ? 'Нет' : stockStatus === 'low' ? 'Мало' : 'В наличии'}
          </Badge>
        </div>

        <div className="grid grid-cols-2 gap-2 mb-3">
          <div className="text-center bg-slate-50 dark:bg-slate-900 rounded-lg p-2">
            <p className="text-lg font-bold text-slate-900 dark:text-white">
              {part.quantity}
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {part.unit || 'шт'}
            </p>
          </div>
          <div className="text-center bg-slate-50 dark:bg-slate-900 rounded-lg p-2">
            <p className="text-lg font-bold text-slate-900 dark:text-white">
              {part.min_quantity ?? '—'}
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400">Мин. запас</p>
          </div>
        </div>

        {part.category && (
          <div className="flex items-center gap-1 mb-1">
            <Wrench size={12} className="text-slate-400" />
            <span className="text-xs text-slate-500 dark:text-slate-400">{part.category}</span>
          </div>
        )}

        {part.location && (
          <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
            📍 {part.location}
          </p>
        )}

        {part.price != null && (
          <p className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
            ${part.price.toFixed(2)}
          </p>
        )}

        {onOrder && isLowStock && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onOrder(part.id);
            }}
            className="w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            Заказать
          </button>
        )}
      </CardBody>
    </Card>
    </div>
  );
}