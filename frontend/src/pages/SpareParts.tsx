import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSpareParts } from '../context/SparePartsContext';
import { Button, PartCard, Modal, Input } from '../components/ui';
import { Plus, Search } from 'lucide-react';

export const SpareParts: React.FC = () => {
  const { t } = useTranslation();
  const { spareParts, loading, createSparePart } = useSpareParts();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [search, setSearch] = useState('');

  const filtered = spareParts.filter((p) => {
    if (search && !p.name.toLowerCase().includes(search.toLowerCase()) && !p.sku.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const mapToPartCard = (p: typeof spareParts[0]) => ({
    id: p.id,
    name: p.name,
    part_number: p.sku,
    category: p.category,
    quantity: p.stock,
    min_quantity: p.min_stock,
    unit: 'шт',
    price: p.cost,
    supplier: p.supplier,
    location: p.location,
  });

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('spare_parts')}</h1>
        <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
          {t('add_part')}
        </Button>
      </div>

      <div className="mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder={t('search_parts')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 border border-slate-300 dark:border-slate-600 rounded-xl bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="animate-pulse bg-white dark:bg-slate-800 rounded-xl p-5 h-48 border border-slate-200 dark:border-slate-700" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((part) => (
            <PartCard key={part.id} part={mapToPartCard(part)} />
          ))}
          {filtered.length === 0 && (
            <div className="col-span-full text-center py-12 text-slate-500 dark:text-slate-400">
              {search ? 'No parts match your search' : t('no_parts')}
            </div>
          )}
        </div>
      )}

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
