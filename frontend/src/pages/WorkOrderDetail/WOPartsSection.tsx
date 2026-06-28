// ═══════════════════════════════════════════════════════════════════════
// WOPartsSection — секция запчастей и стоимости работ в правой колонке
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Package, DollarSign } from 'lucide-react';
import { Card, CardHeader, CardBody, Badge } from '../../components/ui';
import type { WorkOrder, SparePart } from '../../hooks/useApiQuery';
import type { LaborCost } from '../../services/workOrdersApi';

interface WOPartsSectionProps {
  workOrder: WorkOrder;
  spareParts: SparePart[];
  laborCost: LaborCost | null;
}

export const WOPartsSection: React.FC<WOPartsSectionProps> = ({
  workOrder,
  spareParts,
  laborCost,
}) => {
  return (
    <>
      {/* Parts Used (compact) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Package className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="text-sm font-medium">
            Запчасти ({workOrder.parts_used?.length || 0})
          </span>
        </CardHeader>
        <CardBody>
          {workOrder.parts_used?.length > 0 ? (
            <div className="space-y-2">
              {workOrder.parts_used.map((p, i) => {
                const part = spareParts.find(sp => sp.id === p.part_id);
                return (
                  <div key={i} className="flex justify-between items-center p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm">
                    <span className="text-slate-700 dark:text-slate-300 truncate mr-2">
                      {part?.name || p.part_id}
                    </span>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <Badge variant="info" size="sm">{p.quantity} шт</Badge>
                      {part?.cost != null && (
                        <span className="text-xs text-slate-500">{part.cost * p.quantity}₽</span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-2">Нет запчастей</p>
          )}
          {workOrder.total_parts_cost != null && (
            <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm">
              <span className="text-slate-500 dark:text-slate-400">Стоимость запчастей</span>
              <span className="font-medium text-slate-900 dark:text-white">{workOrder.total_parts_cost.toFixed(2)} ₽</span>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Labor Cost (compact) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <DollarSign className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <span className="text-sm font-medium">Стоимость работ</span>
        </CardHeader>
        <CardBody>
          {laborCost ? (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-slate-500 dark:text-slate-400">Часов</span>
                <span className="text-slate-900 dark:text-white font-medium">{laborCost.total_hours.toFixed(1)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-500 dark:text-slate-400">Ставка</span>
                <span className="text-slate-900 dark:text-white font-medium">{laborCost.currency}{laborCost.hourly_rate.toFixed(2)}</span>
              </div>
              <div className="pt-2 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm font-semibold">
                <span className="text-emerald-600 dark:text-emerald-400">Всего</span>
                <span className="text-emerald-700 dark:text-emerald-300">{laborCost.currency}{laborCost.total_cost.toFixed(2)}</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-2">Нет данных</p>
          )}
        </CardBody>
      </Card>
    </>
  );
};
