import type { Meta, StoryObj } from '@storybook/react';
import { PartCard } from './PartCard';

const meta: Meta<typeof PartCard> = {
  title: 'UI/PartCard',
  component: PartCard,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof PartCard>;

// ── In Stock ─────────────────────────────────────────────────────────────

export const InStock: Story = {
  args: {
    part: {
      id: '1',
      name: 'HDD 1TB SATA III',
      part_number: 'HDD-SATA-1TB',
      category: 'Storage',
      quantity: 15,
      min_quantity: 5,
      unit: 'шт',
      price: 89.99,
      supplier: 'TechDistributor Inc.',
      location: 'Склад A, стеллаж 3',
    },
  },
};

// ── Low Stock ────────────────────────────────────────────────────────────

export const LowStock: Story = {
  args: {
    part: {
      id: '2',
      name: 'Fan 80mm 12V',
      part_number: 'FAN-80-12V',
      category: 'Cooling',
      quantity: 3,
      min_quantity: 10,
      unit: 'шт',
      price: 12.50,
      supplier: 'CoolingTech',
      location: 'Склад B, стеллаж 1',
    },
    onOrder: (id) => alert(`Order ${id}`),
  },
};

// ── Out of Stock ─────────────────────────────────────────────────────────

export const OutOfStock: Story = {
  args: {
    part: {
      id: '3',
      name: 'Power Supply 48V/2A',
      part_number: 'PSU-48V-2A',
      category: 'Power',
      quantity: 0,
      min_quantity: 5,
      unit: 'шт',
      price: 45.00,
      supplier: 'PowerSys',
      location: 'Склад C, стеллаж 7',
    },
    onOrder: (id) => alert(`Order ${id}`),
  },
};

// ── Without Image (Minimal) ──────────────────────────────────────────────

export const Minimal: Story = {
  args: {
    part: {
      id: '4',
      name: 'Screw Kit M4',
      quantity: 200,
      min_quantity: 50,
      unit: 'шт',
    },
  },
};

// ── With All Details ─────────────────────────────────────────────────────

export const FullDetails: Story = {
  args: {
    part: {
      id: '5',
      name: 'Camera Lens 4mm',
      part_number: 'LENS-4MM-WDR',
      category: 'Optics',
      quantity: 8,
      min_quantity: 4,
      unit: 'шт',
      price: 156.00,
      supplier: 'OpticsPro GmbH',
      location: 'Склад D, стеллаж 2, секция A',
      last_ordered: '2024-10-15',
    },
    onView: (id) => alert(`View ${id}`),
  },
};

// ── Expensive Part ───────────────────────────────────────────────────────

export const ExpensivePart: Story = {
  args: {
    part: {
      id: '6',
      name: 'NVR Main Board',
      part_number: 'MB-NVR-1040P',
      category: 'Electronics',
      quantity: 2,
      min_quantity: 2,
      unit: 'шт',
      price: 1249.00,
      supplier: 'Manufacturer Direct',
      location: 'Склад A, сейф 2',
    },
    onView: (id) => alert(`View ${id}`),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    part: {
      id: '7',
      name: 'Custom Part',
      part_number: 'CUSTOM-001',
      category: 'Misc',
      quantity: 10,
      min_quantity: 3,
      unit: 'шт',
      price: 25.00,
    },
    onView: (id) => alert(`View ${id}`),
    onOrder: (id) => alert(`Order ${id}`),
  },
};
