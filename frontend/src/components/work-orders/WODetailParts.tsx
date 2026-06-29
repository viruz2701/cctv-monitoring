import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Package, Plus, AlertTriangle, X, DollarSign,
} from '../ui/Icons';
import { Card, CardHeader, CardBody, Badge, Button, PartCard } from '../ui';
import { WorkOrder, PartUsage } from '../../services/workOrdersApi';
import { SparePart } from '../../services/sparePartsApi';

interface WODetailPartsProps {
  workOrder: WorkOrder;
  spareParts: SparePart[];
  selectedPartId: string;
  partQuantity: number;
  partsCostLoading: boolean;
  onSelectedPartIdChange: (id: string) => void;
  onPartQuantityChange: (qty: number) => void;
  onAddPart: () => void;
}

export const WODetailParts: React.FC<WODetailPartsProps> = ({
  workOrder,
  spareParts,
  selectedPartId,
  partQuantity,
  partsCostLoading,
  onSelectedPartIdChange,
  onPartQuantityChange,
  onAddPart,
}) => {
  const { t } = useTranslation();
  const [drawerPart, setDrawerPart] = useState<SparePart | null>(null);

  const isReadOnly = workOrder.status === 'completed' || workOrder.status === 'cancelled';

  const handlePartClick = (partId: string) => {
    const part = spareParts.find(sp => sp.id === partId);
    if (part) setDrawerPart(part);
  };

  return (
    <div className="space-y-6">
      {/* Parts Usage List */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Package className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>
            {t('workOrder.partsUsed') || 'Использованные запчасти'}
            <span className="ml-1 text-sm text-slate-400">
              ({workOrder.parts_used?.length || 0})
            </span>
          </span>
        </CardHeader>
        <CardBody>
          {workOrder.parts_used?.length > 0 ? (
            <div className="space-y-2">
              {workOrder.parts_used.map((p, i) => {
                const part = spareParts.find(sp => sp.id === p.part_id);
                return (
                  <div
                    key={i}
                    onClick={() => handlePartClick(p.part_id)}
                    className="flex justify-between items-center p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-700/50 transition-colors"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Package className="w-4 h-4 text-violet-500 flex-shrink-0" />
                      <span className="text-slate-700 dark:text-slate-300 truncate">
                        {part?.name || p.part_id}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <Badge variant="info">{p.quantity} {t('common.pcs') || 'шт'}</Badge>
                      {part?.cost != null && (
                        <span className="text-xs text-slate-500 dark:text-slate-400">
                          {part.cost * p.quantity}₽
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-4">
              {t('workOrder.noPartsUsed') || 'Запчасти не использованы'}
            </p>
          )}
        </CardBody>
      </Card>

      {/* Add Part Form */}
      {!isReadOnly && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Plus className="w-5 h-5 text-violet-600 dark:text-violet-400" />
            <span>{t('workOrder.addPart') || 'Добавить запчасть (со стоимостью)'}</span>
          </CardHeader>
          <CardBody>
            <div className="space-y-3">
              {/* Part Selector */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.part') || 'Запчасть'}
                </label>
                <select
                  value={selectedPartId}
                  onChange={(e) => onSelectedPartIdChange(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm focus:ring-2 focus:ring-violet-500 focus:border-transparent"
                >
                  <option value="">— {t('workOrder.selectPart') || 'Выберите запчасть'} —</option>
                  {spareParts.map(part => (
                    <option key={part.id} value={part.id}>
                      {part.name} (SKU: {part.sku}) — {part.cost}₽
                    </option>
                  ))}
                </select>
              </div>

              {/* Quantity */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.quantity') || 'Количество'}
                </label>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => onPartQuantityChange(Math.max(1, partQuantity - 1))}
                    className="w-8 h-8 rounded-lg bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600 text-lg font-medium"
                  >
                    −
                  </button>
                  <input
                    type="number"
                    min={1}
                    value={partQuantity}
                    onChange={(e) => onPartQuantityChange(Math.max(1, parseInt(e.target.value) || 1))}
                    className="w-16 text-center px-2 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm focus:ring-2 focus:ring-violet-500 focus:border-transparent"
                  />
                  <button
                    onClick={() => onPartQuantityChange(partQuantity + 1)}
                    className="w-8 h-8 rounded-lg bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600 text-lg font-medium"
                  >
                    +
                  </button>
                </div>
              </div>

              {/* Cost Preview */}
              {selectedPartId && (() => {
                const part = spareParts.find(p => p.id === selectedPartId);
                return part ? (
                  <div className="flex items-center gap-2 p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm">
                    <DollarSign className="w-4 h-4 text-emerald-500" />
                    <span className="text-slate-500 dark:text-slate-400">
                      {t('workOrder.cost') || 'Стоимость'}:
                    </span>
                    <span className="font-medium text-slate-900 dark:text-white">
                      {part.cost}₽ × {partQuantity} = {part.cost * partQuantity}₽
                    </span>
                  </div>
                ) : null;
              })()}

              <Button
                fullWidth
                variant="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={onAddPart}
                loading={partsCostLoading}
                disabled={!selectedPartId}
              >
                {t('workOrder.addPartWithCost') || 'Добавить со стоимостью'}
              </Button>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Total Cost Summary */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <DollarSign className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span>{t('workOrder.totalCost') || 'Стоимость запчастей'}</span>
        </CardHeader>
        <CardBody>
          <div className="text-center">
            <p className="text-2xl font-bold text-slate-900 dark:text-white">
              {workOrder.total_parts_cost ? `${workOrder.total_parts_cost.toFixed(2)} ₽` : '0.00 ₽'}
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('workOrder.totalPartsCost') || 'Общая стоимость использованных запчастей'}
            </p>
          </div>
        </CardBody>
      </Card>

      {/* Drawer — Part Detail */}
      {drawerPart && (
        <>
          {/* Overlay */}
          <div
            className="fixed inset-0 bg-black/40 z-40 transition-opacity"
            onClick={() => setDrawerPart(null)}
          />

          {/* Drawer panel */}
          <div className="fixed inset-y-0 right-0 w-full max-w-md z-50 bg-white dark:bg-slate-900 shadow-2xl transform transition-transform duration-300 ease-in-out overflow-y-auto">
            <div className="sticky top-0 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 px-6 py-4 flex items-center justify-between z-10">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                {drawerPart.name}
              </h3>
              <button
                onClick={() => setDrawerPart(null)}
                className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
              >
                <X className="w-5 h-5 text-slate-500" />
              </button>
            </div>

            <div className="p-6 space-y-6">
              {/* Part Card */}
              <PartCard
                part={{
                  id: drawerPart.id,
                  name: drawerPart.name,
                  part_number: drawerPart.sku,
                  category: drawerPart.category,
                  quantity: drawerPart.stock,
                  min_quantity: drawerPart.min_stock,
                  unit: 'шт',
                  price: drawerPart.cost,
                  supplier: drawerPart.supplier,
                  location: drawerPart.location,
                }}
              />

              {/* Usage history in this work order */}
              <div>
                <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
                  {t('workOrder.usageHistory') || 'История использования'}
                </h4>
                {workOrder.parts_used.filter(p => p.part_id === drawerPart.id).length > 0 ? (
                  <div className="space-y-2">
                    {workOrder.parts_used
                      .filter(p => p.part_id === drawerPart.id)
                      .map((p, i) => (
                        <div
                          key={i}
                          className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg"
                        >
                          <span className="text-sm text-slate-700 dark:text-slate-300">
                            {t('workOrder.used') || 'Использовано'}
                          </span>
                          <Badge variant="info">{p.quantity} {t('common.pcs') || 'шт'}</Badge>
                        </div>
                      ))}
                  </div>
                ) : (
                  <p className="text-sm text-slate-400 dark:text-slate-500 italic">
                    {t('workOrder.noUsageHistory') || 'Нет истории использования'}
                  </p>
                )}
              </div>

              {/* Action */}
              {!isReadOnly && (
                <Button
                  fullWidth
                  variant="primary"
                  icon={<Plus className="w-4 h-4" />}
                  onClick={() => {
                    onSelectedPartIdChange(drawerPart.id);
                    onPartQuantityChange(1);
                    setDrawerPart(null);
                  }}
                >
                  {t('workOrder.addMore') || 'Добавить ещё'}
                </Button>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
