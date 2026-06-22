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
  verification_token?: string;
  location?: {
    latitude: number;
    longitude: number;
  };
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
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
  Verification: { workOrder: WorkOrder; checklist: ChecklistItem[]; photos: string[] };
  Signature: { workOrder: WorkOrder; checklist: ChecklistItem[]; photos: string[]; verificationToken: string };
  QRScanner: undefined;
};

export interface VerificationRequest {
  gps: {
    latitude: number;
    longitude: number;
    accuracy: number;
    timestamp: string;
  };
  photo_exif: {
    gps_latitude: number;
    gps_longitude: number;
    date_time_original: string;
    make: string;
    model: string;
  };
  photo_before_url: string;
  photo_after_url: string;
  checklist_completed: boolean;
  signature: string;
  gps_skip_reason?: string;
}

export interface VerificationResponse {
  passed: boolean;
  token?: string;
  gps: {
    passed: boolean;
    distance_meters: number;
    accuracy_meters: number;
    within_geofence: boolean;
    timestamp_valid: boolean;
    error?: string;
  };
  exif: {
    passed: boolean;
    gps_match: boolean;
    timestamp_valid: boolean;
    has_exif: boolean;
    error?: string;
  };
  ai: {
    passed: boolean;
    similarity: number;
    change_detected: boolean;
    summary?: string;
    error?: string;
    skipped: boolean;
  };
  message?: string;
  fail_reasons?: string[];
}

export type MainTabParamList = {
  Dashboard: undefined;
  Profile: undefined;
};