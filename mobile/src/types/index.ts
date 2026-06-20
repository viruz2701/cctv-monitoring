export interface User {
  id: string;
  username: string;
  role: string;
  email?: string;
}

export interface ChecklistItem {
  task: string;
  completed: boolean;
}

export interface PartUsage {
  part_id: string;
  part_name: string;
  quantity: number;
}

export interface WorkOrder {
  id: string;
  schedule_id?: string;
  device_id: string;
  device_name?: string;
  site_name?: string;
  type: 'preventive' | 'corrective' | 'emergency';
  status: 'open' | 'in_progress' | 'completed' | 'cancelled';
  priority: 'critical' | 'high' | 'medium' | 'low';
  assigned_to?: string;
  sla_deadline?: string;
  checklist: ChecklistItem[];
  started_at?: string;
  completed_at?: string;
  notes?: string;
  photos: string[];
  parts_used: PartUsage[];
  created_by?: string;
  created_at: string;
  updated_at: string;
  device_name_display?: string;
  assignee_name?: string;
  sla_status?: string;
}

export interface CompleteWorkOrderPayload {
  notes: string;
  checklist: ChecklistItem[];
  photos: string[];
  parts_used: PartUsage[];
  signature?: string;
  location?: {
    latitude: number;
    longitude: number;
  };
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface TechnicianProfile {
  user_id: string;
  user_name: string;
  current_workload: number;
  max_workload: number;
  skills: string[];
  base_location: string;
}

export interface TechnicianStats {
  completed_this_month: number;
  total_work_orders: number;
  on_time_percent: number;
  avg_rating: number;
}

export type RootStackParamList = {
  Login: undefined;
  Main: undefined;
  WorkOrderDetail: { workOrderId: string };
  Checklist: { workOrder: WorkOrder };
  PhotoCapture: { workOrder: WorkOrder; checklist: ChecklistItem[] };
  Signature: { workOrder: WorkOrder; checklist: ChecklistItem[]; photos: string[] };
  QRScanner: undefined;
};

export type MainTabParamList = {
  Dashboard: undefined;
  Profile: undefined;
};