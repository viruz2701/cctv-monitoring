import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSpareParts } from '../context/SparePartsContext';
import { SparePart } from '../services/sparePartsApi';
import { Button, Card, Table, Modal, Input } from '../components/ui';
import { Plus, AlertTriangle, Minus, Package } from 'lucide-react';

export const SpareParts: React.FC = () => {
  const { t } = useTranslation();
  const { spareParts, loading, deleteSparePart, adjustStock } = useSpareParts();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [search, setSearch] = useState('');

  const filtered = spareParts.filter((p) => {
    if (search && !p.name.toLowerCase().includes(search.toLowerCase()) && !p.sku.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const isLowStock = (part: SparePart) => part.stock <= part.min_stock;

  const columns = [
    {
      key: 'name',
      header: t('name'),
      render: (item: SparePart) => (
        <div className="flex items-center gap-2">
          <Package size={16} className="text-slate-400" />
          {item.name}
        </div>
      ),
    },
    { key: 'sku', header: 'SKU' },
    { key: 'category', header: t('category'), render: (item: SparePart) => item.category || '-' },
    {
      key: 'stock',
      header: t('stock'),
      render: (item: SparePart) => (
        <div className="flex items-center gap-2">
          <span className={isLowStock(item) ? 'text-red-600 font-bold' : ''}>
            {item.stock}
          </span>
          {isLowStock(item) && <AlertTriangle className="text-red-500" size={16} />}
          <span className="text-xs text-slate-400">/ min: {item.min_stock}</span>
        </div>
      ),
    },
    {
      key: 'cost',
      header: t('cost'),
      render: (item: SparePart) => `$${item.cost?.toFixed(2) || '0.00'}`,
    },
    { key: 'location', header: t('location'), render: (item: SparePart) => item.location || '-' },
    {
      key: 'actions',
      header: t('actions'),
      render: (item: SparePart) => (
        <div className="flex gap-1">
          <Button size="sm" onClick={(e) => { e.stopPropagation(); adjustStock(item.id, item.stock + 1); }} icon={<Plus size={14} />} />
          <Button size="sm" variant="secondary" onClick={(e) => { e.stopPropagation(); adjustStock(item.id, Math.max(0, item.stock - 1)); }} icon={<Minus size={14} />} />
          <Button size="sm" variant="danger" onClick={(e) => { e.stopPropagation(); deleteSparePart(item.id); }}>
            {t('delete')}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('spare_parts')}</h1>
        <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
          {t('add_part')}
        </Button>
      </div>

      <Card>
        <div className="mb-4">
          <Input
            placeholder={t('search_parts')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Table
          data={filtered}
          columns={columns}
          keyExtractor={(item) => item.id}
          loading={loading}
          emptyMessage={t('no_parts')}
        />
      </Card>

      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title={t('add_part')}>
        <CreatePartForm onClose={() => setShowCreateModal(false)} />
      </Modal>
    </div>
  );
};

const CreatePartForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const { createSparePart } = useSpareParts();
  const [formData, setFormData] = useState({
    name: '',
    sku: '',
    category: '',
    stock: 0,
    min_stock: 5,
    location: '',
    cost: 0,
    supplier: '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createSparePart(formData);
    onClose();
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input label={t('name')} value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required />
      <Input label="SKU" value={formData.sku} onChange={(e) => setFormData({ ...formData, sku: e.target.value })} />
      <Input label={t('category')} value={formData.category} onChange={(e) => setFormData({ ...formData, category: e.target.value })} />
      <div className="grid grid-cols-2 gap-4">
        <Input type="number" label={t('stock')} value={String(formData.stock)} onChange={(e) => setFormData({ ...formData, stock: parseInt(e.target.value) || 0 })} />
        <Input type="number" label={t('min_stock')} value={String(formData.min_stock)} onChange={(e) => setFormData({ ...formData, min_stock: parseInt(e.target.value) || 0 })} />
      </div>
      <Input type="number" label={t('cost')} value={String(formData.cost)} onChange={(e) => setFormData({ ...formData, cost: parseFloat(e.target.value) || 0 })} />
      <Input label={t('location')} value={formData.location} onChange={(e) => setFormData({ ...formData, location: e.target.value })} />
      <Input label={t('supplier')} value={formData.supplier} onChange={(e) => setFormData({ ...formData, supplier: e.target.value })} />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>{t('cancel')}</Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};
