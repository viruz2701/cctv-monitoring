// ──────────────────────────────────────────────────
// Mock-данные для E2E тестов Detox
// ──────────────────────────────────────────────────

import { WorkOrder } from '../../src/types';

// ── Work Orders ───────────────────────────────────

export const mockWorkOrders: WorkOrder[] = [
  {
    id: 'wo-001',
    device_id: 'cam-101',
    device_name: 'Camera Main Entrance',
    site_name: 'Facility A',
    type: 'preventive',
    status: 'open',
    priority: 'high',
    assigned_to: 'tech-1',
    sla_deadline: new Date(Date.now() + 86400000).toISOString(),
    checklist: [
      { task: 'Check lens cleanliness', completed: false },
      { task: 'Verify recording status', completed: false },
    ],
    photos: [],
    parts_used: [],
    created_by: 'dispatcher-1',
    created_at: new Date(Date.now() - 172800000).toISOString(),
    updated_at: new Date(Date.now() - 86400000).toISOString(),
    device_name_display: 'CAM-101 — Main Entrance',
    assignee_name: 'John Technician',
    sla_status: 'ok',
  },
  {
    id: 'wo-002',
    device_id: 'cam-102',
    device_name: 'Camera Parking Lot',
    site_name: 'Facility A',
    type: 'emergency',
    status: 'in_progress',
    priority: 'critical',
    assigned_to: 'tech-1',
    sla_deadline: new Date(Date.now() + 3600000).toISOString(),
    checklist: [
      { task: 'Inspect cable damage', completed: true },
      { task: 'Test night vision', completed: false },
    ],
    photos: [],
    parts_used: [{ part_id: 'prt-01', part_name: 'Ethernet cable 10m', quantity: 1 }],
    created_by: 'dispatcher-1',
    created_at: new Date(Date.now() - 7200000).toISOString(),
    updated_at: new Date(Date.now() - 3600000).toISOString(),
    device_name_display: 'CAM-102 — Parking Lot',
    assignee_name: 'John Technician',
    sla_status: 'warning',
  },
];

// ── Work Order (server-side fresher) for conflict ──

export const mockServerWorkOrder: WorkOrder = {
  ...mockWorkOrders[0],
  status: 'in_progress',
  updated_at: new Date(Date.now() - 3600000).toISOString(),
  notes: 'Started by dispatcher remotely',
};

// ── Work Order (local changed) for conflict ──────

export const mockLocalWorkOrder: WorkOrder = {
  ...mockWorkOrders[0],
  status: 'completed',
  updated_at: new Date(Date.now() - 1800000).toISOString(),
  notes: 'Completed by technician on site',
  checklist: [
    { task: 'Check lens cleanliness', completed: true },
    { task: 'Verify recording status', completed: true },
  ],
};

// ── Gatekeeper Verification Responses ────────────

export const mockVerificationResponse = {
  passed: true,
  token: 'vrf_tkn_abc123',
  gps: {
    passed: true,
    distance_meters: 2.5,
    accuracy_meters: 5.0,
    within_geofence: true,
    timestamp_valid: true,
  },
  exif: {
    passed: true,
    gps_match: true,
    timestamp_valid: true,
    has_exif: true,
  },
  ai: {
    passed: true,
    similarity: 0.97,
    change_detected: true,
    summary: 'Work verified — all checks passed',
    skipped: false,
    error: undefined,
  },
};

export const mockVerificationGpsFailed = {
  passed: false,
  token: undefined,
  gps: {
    passed: false,
    distance_meters: 150.0,
    accuracy_meters: 50.0,
    within_geofence: false,
    timestamp_valid: true,
    error: 'GPS location is outside the geofence (150m > 50m threshold)',
  },
  exif: {
    passed: true,
    gps_match: false,
    timestamp_valid: true,
    has_exif: true,
  },
  ai: {
    passed: true,
    similarity: 0.95,
    change_detected: true,
    summary: 'Work completed, but GPS location mismatch',
    skipped: false,
    error: undefined,
  },
};

export const mockVerificationExifFailed = {
  passed: false,
  token: undefined,
  gps: {
    passed: true,
    distance_meters: 3.0,
    accuracy_meters: 4.0,
    within_geofence: true,
    timestamp_valid: true,
  },
  exif: {
    passed: false,
    gps_match: false,
    timestamp_valid: false,
    has_exif: false,
    error: 'EXIF data missing — photo may be edited',
  },
  ai: {
    passed: false,
    similarity: 0.45,
    change_detected: false,
    summary: 'Work verification failed — EXIF data mismatch',
    skipped: false,
    error: undefined,
  },
};

// ── Auth ──────────────────────────────────────────

export const mockLoginResponse = {
  token: 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.mock-token',
  refresh_token: 'mock-refresh-token-abc',
  user: {
    id: 'tech-1',
    username: 'johntech',
    role: 'technician',
    email: 'john.tech@cctv-monitoring.com',
  },
};

export const mockProfileResponse = {
  user_id: 'tech-1',
  user_name: 'John Technician',
  current_workload: 3,
  max_workload: 8,
  skills: ['cctv', 'networking', 'power-systems'],
  base_location: 'Minsk, Belarus',
};

// ── Device Map ────────────────────────────────────

export const mockDevicesForMap = {
  devices: [
    {
      device_id: 'cam-101',
      name: 'Camera Main Entrance',
      device_type: 'bullet',
      status: 'ONLINE',
      site_name: 'Facility A',
      latitude: 53.893,
      longitude: 27.555,
      health: 'healthy',
    },
    {
      device_id: 'cam-102',
      name: 'Camera Parking Lot',
      device_type: 'ptz',
      status: 'DEGRADED',
      site_name: 'Facility A',
      latitude: 53.895,
      longitude: 27.558,
      health: 'degraded',
    },
  ],
};
