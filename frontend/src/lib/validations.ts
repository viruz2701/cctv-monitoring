import { z } from 'zod';

// ─── Device ───────────────────────────────────────────────────────────────
export const deviceSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(100, 'Name must be at most 100 characters'),
  ipAddress: z.string().ip('Invalid IP address'),
  siteId: z.string().uuid('Invalid site ID'),
  type: z.enum(['camera', 'nvr', 'dvr', 'switch'], { message: 'Invalid device type' }),
  model: z.string().optional(),
});

export type DeviceFormData = z.infer<typeof deviceSchema>;

// ─── Site ─────────────────────────────────────────────────────────────────
export const siteSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(200, 'Name must be at most 200 characters'),
  address: z.string().min(5, 'Address must be at least 5 characters').max(300, 'Address must be at most 300 characters'),
  city: z.string().min(2, 'City must be at least 2 characters').max(100, 'City must be at most 100 characters'),
  status: z.enum(['active', 'inactive', 'maintenance'], { message: 'Invalid status' }),
  organization: z.string().optional(),
});

export type SiteFormData = z.infer<typeof siteSchema>;

// ─── Ticket ───────────────────────────────────────────────────────────────
export const ticketSchema = z.object({
  title: z.string().min(5, 'Title must be at least 5 characters').max(200, 'Title must be at most 200 characters'),
  description: z.string().min(10, 'Description must be at least 10 characters').max(2000, 'Description must be at most 2000 characters'),
  priority: z.enum(['critical', 'high', 'medium', 'low'], { message: 'Invalid priority' }),
  deviceId: z.string().uuid('Invalid device ID').optional(),
  siteId: z.string().optional(),
});

export type TicketFormData = z.infer<typeof ticketSchema>;

// ─── User ─────────────────────────────────────────────────────────────────
export const userSchema = z.object({
  username: z.string().min(3, 'Username must be at least 3 characters').max(50, 'Username must be at most 50 characters'),
  email: z.string().email('Invalid email address'),
  role: z.enum(['admin', 'manager', 'technician', 'viewer'], { message: 'Invalid role' }),
  password: z.string().min(6, 'Password must be at least 6 characters').optional().or(z.literal('')),
});

export type UserFormData = z.infer<typeof userSchema>;

// ─── Work Order ───────────────────────────────────────────────────────────
export const workOrderSchema = z.object({
  title: z.string().min(5, 'Title must be at least 5 characters').max(200, 'Title must be at most 200 characters'),
  description: z.string().min(10, 'Description must be at least 10 characters').max(5000, 'Description must be at most 5000 characters'),
  priority: z.enum(['critical', 'high', 'medium', 'low'], { message: 'Invalid priority' }),
  type: z.enum(['preventive', 'corrective', 'emergency'], { message: 'Invalid work order type' }),
  deviceId: z.string().uuid('Invalid device ID'),
  assignedTo: z.string().uuid('Invalid user ID').optional().or(z.literal('')),
});

export type WorkOrderFormData = z.infer<typeof workOrderSchema>;

// ─── API Key ──────────────────────────────────────────────────────────────
export const apiKeySchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters').max(100, 'Name must be at most 100 characters'),
  permissions: z.array(z.enum(['read', 'write', 'admin'])).min(1, 'At least one permission is required'),
});

export type ApiKeyFormData = z.infer<typeof apiKeySchema>;

// ─── Profile ──────────────────────────────────────────────────────────────
export const profileSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  email: z.string().email('Invalid email address'),
  phone: z.string().min(10, 'Phone must be at least 10 digits').optional().or(z.literal('')),
  location: z.string().max(100, 'Location is too long').optional(),
});

export type ProfileFormData = z.infer<typeof profileSchema>;
