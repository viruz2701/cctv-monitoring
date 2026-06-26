// notifications — Push-уведомления для критических событий.
//
// P1-3.3: Push Notifications for SLA Breach
//   - Интеграция с expo-notifications
//   - Deep linking в WorkOrderDetail
//   - User preferences для типов уведомлений

import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

export type NotificationType =
  | 'sla_breach'
  | 'alarm'
  | 'work_order'
  | 'maintenance';

export interface NotificationPreferences {
  sla_breach: boolean;
  alarm_critical: boolean;
  alarm_high: boolean;
  work_order_assigned: boolean;
  maintenance_due: boolean;
}

// ──────────────────────────────────────────────────
// Handler configuration
// ──────────────────────────────────────────────────

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

// ──────────────────────────────────────────────────
// Registration
// ──────────────────────────────────────────────────

export async function registerForPushNotifications(): Promise<string | null> {
  const { status: existingStatus } = await Notifications.getPermissionsAsync();
  let finalStatus = existingStatus;

  if (existingStatus !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }

  if (finalStatus !== 'granted') {
    return null;
  }

  const tokenData = await Notifications.getExpoPushTokenAsync();
  const token = tokenData.data;

  if (Platform.OS === 'android') {
    await Notifications.setNotificationChannelAsync('default', {
      name: 'Default',
      importance: Notifications.AndroidImportance.MAX,
      vibrationPattern: [0, 250, 250, 250],
      lightColor: '#FF231F7C',
    });
  }

  return token;
}

// ──────────────────────────────────────────────────
// Local notifications
// ──────────────────────────────────────────────────

export async function sendLocalNotification(
  title: string,
  body: string,
  data?: Record<string, unknown>,
): Promise<void> {
  await Notifications.scheduleNotificationAsync({
    content: {
      title,
      body,
      data: data || {},
      sound: true,
    },
    trigger: null, // immediate
  });
}

// ──────────────────────────────────────────────────
// Deep linking
// ──────────────────────────────────────────────────

export function createDeepLink(workOrderId: string): string {
  return `cctv-monitor://work-order/${workOrderId}`;
}
